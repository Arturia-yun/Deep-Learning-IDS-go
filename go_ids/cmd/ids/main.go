package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-ids/internal/capture"
	"go-ids/internal/db"
	"go-ids/internal/decoder"
	"go-ids/internal/feature"
	"go-ids/internal/flow"
	"go-ids/internal/inference"
	"go-ids/internal/loader"
	"go-ids/internal/logger"
	"go-ids/internal/response"
	"go-ids/internal/server"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/sirupsen/logrus"
)

func main() {
	// 1. 解析命令行参数
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	// 2. 加载配置
	cfg, err := loader.Load(*configPath)
	if err != nil {
		logrus.Fatalf("加载配置失败: %v", err)
	}

	// 3. 设置日志
	// 3. 设置日志
	logger.Setup(cfg.Logging)
	logrus.Info("Go Deep-Learning IDS 正在启动...")

	// 4. 初始化数据库
	if err := db.InitDB("config/ids.db"); err != nil {
		logrus.Fatalf("初始化数据库失败: %v", err)
	}
	logrus.Info("SQLite 数据库初始化成功")

	// 5. 启动 Web Server (Gin)
	go func() {
		logrus.Info("启动 Web Server on :8080")
		if err := server.StartServer("8080"); err != nil {
			logrus.Errorf("Web Server 启动失败: %v", err)
		}
	}()

	// 6. 初始化推理引擎
	engine, err := inference.NewEngine(cfg.Detection.ModelPath, cfg.Detection.ORTLibPath)
	if err != nil {
		logrus.Fatalf("初始化推理引擎失败: %v", err)
	}
	defer engine.Close()
	logrus.Info("ONNX 推理引擎初始化成功")

	// 5. 初始化特征提取相关组件
	scaler, err := feature.NewScaler(cfg.Detection.ScalerPath)
	if err != nil {
		logrus.Fatalf("初始化标准化器失败: %v", err)
	}
	extractor := feature.NewExtractor()

	// 7. 初始化流管理器
	// 使用配置中的超时时间
	flowMgr := flow.NewManager(time.Duration(cfg.Flow.TCPTimeout) * time.Second)
	// 注入到 Web Server 以展示活跃连接数
	server.SetFlowCounter(flowMgr)

	// 7. 初始化响应器
	responder := response.NewResponder(
		cfg.Response.EnableBlock,
		cfg.Response.BlockDuration,
		cfg.Response.Whitelist,
	)

	// 8. 初始化捕获和解码
	pktSource, err := capture.NewPcapSource(
		cfg.Capture.Interface,
		int32(cfg.Capture.Snaplen),
		cfg.Capture.Promiscuous,
	)
	if err != nil {
		logrus.Errorf("无法打开捕获设备 %s: %v (已切换至仅Web模式)", cfg.Capture.Interface, err)
	} else {
		defer pktSource.Close()
	}

	pktDecoder := decoder.NewDecoder()

	// 9. 启动后台清理与检测协程
	stopChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.Flow.CleanupInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 清理过期流并执行检测
				expiredFlows := flowMgr.Cleanup()
				if len(expiredFlows) > 0 {
					logrus.Debugf("清理并分析 %d 个过期流", len(expiredFlows))
					for _, f := range expiredFlows {
						// 1. 提取原始特征
						rawFeatures := extractor.Extract(f)
						// 2. 特征标准化
						scaledFeatures, err := scaler.Transform(rawFeatures)
						if err != nil {
							logrus.Errorf("特征标准化失败: %v", err)
							continue
						}
						// 3. 推理预测
						pred, err := engine.Predict(scaledFeatures)
						if err != nil {
							logrus.Errorf("推理失败: %v", err)
							continue
						}

						// 4. 响应处理
						currentThreshold := loader.GetConfig().Detection.Threshold
						if pred.Label != "Benign" && float64(pred.Probability) >= currentThreshold {
							event := response.Event{
								SourceIP:   f.Key.SrcIP,
								DestIP:     f.Key.DstIP,
								Label:      pred.Label,
								Confidence: pred.Probability,
								Timestamp:  time.Now(),
								Payload:    string(f.RawPayload), // 提取并转换 Payload
							}
							responder.Handle(event)
						}
					}
				}
			case <-stopChan:
				return
			}
		}
	}()

	// 10. 处理退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logrus.Infof("开始在接口 %s 上监听流量...", cfg.Capture.Interface)

	// 11. 解析家庭网络CIDR
	var homeNets []*net.IPNet
	for _, cidr := range cfg.Networks.HomeNet {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			homeNets = append(homeNets, ipNet)
		}
	}

	// 12. 主数据包处理循环
	var packets <-chan gopacket.Packet
	if pktSource != nil {
		packets = pktSource.Packets()
	}

	isHomeNet := func(ipStr string) bool {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return false
		}
		for _, net := range homeNets {
			if net.Contains(ip) {
				return true
			}
		}
		return false
	}

	for {
		select {
		case <-sigChan:
			logrus.Info("接收到停止信号，正在退出...")
			close(stopChan)
			return
		case packet := <-packets:
			if packet == nil {
				continue
			}

			// 解码包
			decoded, err := pktDecoder.Decode(packet)
			if err != nil || decoded == nil {
				continue
			}

			// 流量统计 logic
			length := packet.Metadata().CaptureLength
			srcHome := isHomeNet(decoded.SrcIP)
			dstHome := isHomeNet(decoded.DstIP)

			// Upload: Src is Home (Outgoing)
			// Download: Dst is Home (Incoming)
			// If both Home -> Internal (Count as both or pick one? Let's count as both for total throughput viz)
			// If neither -> Transit (Count as In?)
			// Simple logic:
			inVal, outVal := 0, 0

			if srcHome {
				outVal = length // We sent it
			}
			if dstHome {
				inVal = length // We received it
			}

			// Transit traffic fallback (e.g. bridging)
			if !srcHome && !dstHome {
				inVal = length // Assume everything foreign is incoming if we see it? Or just ignore direction.
			}

			server.AddTraffic(inVal, outVal)

			// 创建流键
			key := flow.FlowKey{
				SrcIP:   decoded.SrcIP,
				DstIP:   decoded.DstIP,
				SrcPort: decoded.SrcPort,
				DstPort: decoded.DstPort,
				Proto:   layers.IPProtocol(decoded.Protocol),
			}

			// 获取或创建流，并更新状态
			f, isForward := flowMgr.GetOrCreate(key, packet)
			f.Update(packet, isForward)
		}
	}
}
