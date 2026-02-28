package response

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"go-ids/internal/db"
	"go-ids/internal/server"

	"github.com/sirupsen/logrus"
)

// Event 描述一个检测到的威胁事件
type Event struct {
	SourceIP   string
	DestIP     string
	Label      string
	Confidence float32
	Timestamp  time.Time
	Payload    string // 新增: 攻击报文 Hex 或明文
}

// Responder 负责处理威胁事件
type Responder struct {
	enableBlock   bool
	blockDuration time.Duration
	whitelist     map[string]bool
	blockedIPs    map[string]time.Time
	mu            sync.Mutex
}

// NewResponder 创建一个新的响应器
func NewResponder(enableBlock bool, blockDurationSeconds int, whitelist []string) *Responder {
	wlMap := make(map[string]bool)
	for _, ip := range whitelist {
		wlMap[ip] = true
	}

	return &Responder{
		enableBlock:   enableBlock,
		blockDuration: time.Duration(blockDurationSeconds) * time.Second,
		whitelist:     wlMap,
		blockedIPs:    make(map[string]time.Time),
	}
}

// Handle 处理威胁事件
func (r *Responder) Handle(event Event) {
	// 1. 记录日志
	logrus.WithFields(logrus.Fields{
		"src":        event.SourceIP,
		"dst":        event.DestIP,
		"type":       event.Label,
		"confidence": fmt.Sprintf("%.2f", event.Confidence),
	}).Warn("检测到入侵威胁!")

	// 2. 如果是合法流量，直接跳过
	if event.Label == "Benign" {
		return
	}

	// 3. 检查白名单
	if r.whitelist[event.SourceIP] {
		logrus.Infof("IP %s 在白名单中，忽略封禁操作", event.SourceIP)
		return
	}

	// 4. 保存到数据库
	alert := &db.Alert{
		CreatedAt:  time.Now(),
		SourceIP:   event.SourceIP,
		DestIP:     event.DestIP,
		Type:       event.Label,
		Confidence: event.Confidence,
		Payload:    event.Payload, // 存入载荷
	}
	if err := db.CreateAlert(alert); err != nil {
		logrus.Errorf("保存报警信息失败: %v", err)
	}

	// 5. 通过 SSE 推送给前端
	select {
	case server.Manager.Message <- *alert:
	default:
		// 防止阻塞
	}

	// 6. 执行封禁逻辑
	if r.enableBlock {
		r.blockIP(event.SourceIP)
	}
}

func (r *Responder) blockIP(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 如果已经在封禁名单中，更新封禁时间
	if _, exists := r.blockedIPs[ip]; exists {
		r.blockedIPs[ip] = time.Now()
		return
	}

	logrus.Errorf("正在封禁恶意源 IP: %s", ip)

	// 根据操作系统执行不同的封禁命令
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		// 使用 iptables 封禁 (需 root 权限)
		cmd = exec.Command("iptables", "-A", "INPUT", "-s", ip, "-j", "DROP")
	case "windows":
		// 使用 netsh 封禁 (需管理员权限)
		cmd = exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
			"name=IDS_BLOCK_"+ip, "dir=in", "action=block", "remoteip="+ip)
	default:
		logrus.Warnf("当前系统 %s 不支持自动封禁功能", runtime.GOOS)
		return
	}

	if err := cmd.Run(); err != nil {
		logrus.Errorf("执行封禁命令失败: %v", err)
	} else {
		r.blockedIPs[ip] = time.Now()
		logrus.Infof("成功封禁 IP: %s", ip)

		// 启动定时器，到期自动解封
		if r.blockDuration > 0 {
			go r.unblockAfter(ip, r.blockDuration)
		}
	}
}

func (r *Responder) unblockAfter(ip string, delay time.Duration) {
	time.Sleep(delay)
	r.unblockIP(ip)
}

func (r *Responder) unblockIP(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.blockedIPs[ip]; !exists {
		return
	}

	logrus.Infof("正在解除封禁 IP: %s", ip)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("iptables", "-D", "INPUT", "-s", ip, "-j", "DROP")
	case "windows":
		cmd = exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name=IDS_BLOCK_"+ip)
	}

	if cmd != nil {
		if err := cmd.Run(); err != nil {
			logrus.Errorf("解除封禁命令执行失败: %v", err)
		} else {
			delete(r.blockedIPs, ip)
			logrus.Infof("成功解除 IP 封禁: %s", ip)
		}
	}
}

// IsWhitelisted 检查 IP 是否在白名单
func (r *Responder) IsWhitelisted(ip string) bool {
	return r.whitelist[ip]
}
