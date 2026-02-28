package flow

import (
	"math"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Flow 存储单个网络流的状态和统计信息
// 这些统计信息最终将被转换为 78 个输入特征
type Flow struct {
	Key       FlowKey
	StartTime time.Time
	LastTime  time.Time

	// 基础统计
	FwdPackets uint64
	BwdPackets uint64
	FwdBytes   uint64
	BwdBytes   uint64

	// 数据包长度统计 (Fwd)
	FwdPktLenMax   float64
	FwdPktLenMin   float64
	FwdPktLenSum   float64
	FwdPktLenSqSum float64 // 用于计算标准差

	// 数据包长度统计 (Bwd)
	BwdPktLenMax   float64
	BwdPktLenMin   float64
	BwdPktLenSum   float64
	BwdPktLenSqSum float64

	// 总体数据包长度统计
	PktLenMax   float64
	PktLenMin   float64
	PktLenSum   float64
	PktLenSqSum float64

	// 到达时间统计 (IAT)
	FlowIATMax   float64
	FlowIATMin   float64
	FlowIATSum   float64
	FlowIATSqSum float64

	// Fwd IAT
	LastFwdTime time.Time
	FwdIATMax   float64
	FwdIATMin   float64
	FwdIATSum   float64
	FwdIATSqSum float64

	// Bwd IAT
	LastBwdTime time.Time
	BwdIATMax   float64
	BwdIATMin   float64
	BwdIATSum   float64
	BwdIATSqSum float64

	// TCP 标志位计数
	FwdPSHFlags uint32
	BwdPSHFlags uint32
	FwdURGFlags uint32
	BwdURGFlags uint32

	FINFlagCount uint32
	SYNFlagCount uint32
	RSTFlagCount uint32
	PSHFlagCount uint32
	ACKFlagCount uint32
	URGFlagCount uint32
	CWEFlagCount uint32
	ECEFlagCount uint32

	// 首部长度
	FwdHeaderLen uint64
	BwdHeaderLen uint64

	// TCP 窗口与段大小
	InitWinBytesFwd uint32
	InitWinBytesBwd uint32
	FwdActDataPkts  uint32
	FwdMinSegSize   uint32

	// Active/Idle 统计 (简化实现)
	ActiveSum   float64
	ActiveMax   float64
	ActiveMin   float64
	ActiveSqSum float64
	IdleSum     float64
	IdleMax     float64
	IdleMin     float64
	IdleSqSum   float64

	lastFlowPktTime time.Time

	// 攻击审计: 缓存前 N 个包的应用层 Payload
	RawPayload []byte
	pktCount   int
}

// NewFlow 初始化一个新的流
func NewFlow(key FlowKey, pkt gopacket.Packet) *Flow {
	now := pkt.Metadata().Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	f := &Flow{
		Key:       key,
		StartTime: now,
		LastTime:  now,
		// 初始化极值
		FwdPktLenMin: 1e9,
		BwdPktLenMin: 1e9,
		PktLenMin:    1e9,
		FlowIATMin:   1e9,
		FwdIATMin:    1e9,
		BwdIATMin:    1e9,
		ActiveMin:    1e9,
		IdleMin:      1e9,

		lastFlowPktTime: now,
	}

	return f
}

// Update 根据新到达的数据包更新流状态
func (f *Flow) Update(pkt gopacket.Packet, isForward bool) {
	now := pkt.Metadata().Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	// 计算 Flow IAT
	iat := now.Sub(f.lastFlowPktTime).Seconds() * 1000000 // 微秒
	if f.FwdPackets+f.BwdPackets > 0 {
		if iat > f.FlowIATMax {
			f.FlowIATMax = iat
		}
		if iat < f.FlowIATMin {
			f.FlowIATMin = iat
		}
		f.FlowIATSum += iat
		f.FlowIATSqSum += iat * iat
	}
	f.lastFlowPktTime = now
	f.LastTime = now

	// 提取应用层 Payload (最多缓存前 10 个包，总计不超过 4KB)
	f.pktCount++
	if f.pktCount <= 10 && len(f.RawPayload) < 4096 {
		if appLayer := pkt.ApplicationLayer(); appLayer != nil {
			payload := appLayer.Payload()
			if len(payload) > 0 {
				f.RawPayload = append(f.RawPayload, payload...)
				if len(f.RawPayload) > 4096 {
					f.RawPayload = f.RawPayload[:4096]
				}
			}
		}
	}

	// 提取包长
	pktLen := float64(pkt.Metadata().Length)
	if pktLen > f.PktLenMax {
		f.PktLenMax = pktLen
	}
	if pktLen < f.PktLenMin {
		f.PktLenMin = pktLen
	}
	f.PktLenSum += pktLen
	f.PktLenSqSum += pktLen * pktLen

	// 处理 TCP 特定信息
	var tcp *layers.TCP
	if tcpLayer := pkt.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp = tcpLayer.(*layers.TCP)
		f.updateTCPFlags(tcp)
		if isForward && f.FwdPackets == 0 {
			f.InitWinBytesFwd = uint32(tcp.Window)
		} else if !isForward && f.BwdPackets == 0 {
			f.InitWinBytesBwd = uint32(tcp.Window)
		}
	}

	// 处理首部长度 (IP + Transport)
	headerLen := uint64(0)
	if ip4 := pkt.Layer(layers.LayerTypeIPv4); ip4 != nil {
		headerLen += uint64(ip4.(*layers.IPv4).IHL) * 4
	}
	if tcp != nil {
		headerLen += uint64(tcp.DataOffset) * 4
		if isForward {
			if f.FwdMinSegSize == 0 || uint32(tcp.DataOffset)*4 < f.FwdMinSegSize {
				f.FwdMinSegSize = uint32(tcp.DataOffset) * 4
			}
		}
	} else if udp := pkt.Layer(layers.LayerTypeUDP); udp != nil {
		headerLen += 8
	}

	if isForward {
		f.FwdPackets++
		f.FwdBytes += uint64(pktLen)
		f.FwdHeaderLen += headerLen
		if pktLen > f.FwdPktLenMax {
			f.FwdPktLenMax = pktLen
		}
		if pktLen < f.FwdPktLenMin {
			f.FwdPktLenMin = pktLen
		}
		f.FwdPktLenSum += pktLen
		f.FwdPktLenSqSum += pktLen * pktLen

		if !f.LastFwdTime.IsZero() {
			fiat := now.Sub(f.LastFwdTime).Seconds() * 1000000
			if fiat > f.FwdIATMax {
				f.FwdIATMax = fiat
			}
			if fiat < f.FwdIATMin {
				f.FwdIATMin = fiat
			}
			f.FwdIATSum += fiat
			f.FwdIATSqSum += fiat * fiat
		}
		f.LastFwdTime = now

		if pktLen > 0 {
			f.FwdActDataPkts++
		}
		if tcp != nil && tcp.PSH {
			f.FwdPSHFlags++
		}
		if tcp != nil && tcp.URG {
			f.FwdURGFlags++
		}

	} else {
		f.BwdPackets++
		f.BwdBytes += uint64(pktLen)
		f.BwdHeaderLen += headerLen
		if pktLen > f.BwdPktLenMax {
			f.BwdPktLenMax = pktLen
		}
		if pktLen < f.BwdPktLenMin {
			f.BwdPktLenMin = pktLen
		}
		f.BwdPktLenSum += pktLen
		f.BwdPktLenSqSum += pktLen * pktLen

		if !f.LastBwdTime.IsZero() {
			biat := now.Sub(f.LastBwdTime).Seconds() * 1000000
			if biat > f.BwdIATMax {
				f.BwdIATMax = biat
			}
			if biat < f.BwdIATMin {
				f.BwdIATMin = biat
			}
			f.BwdIATSum += biat
			f.BwdIATSqSum += biat * biat
		}
		f.LastBwdTime = now

		if tcp != nil && tcp.PSH {
			f.BwdPSHFlags++
		}
		if tcp != nil && tcp.URG {
			f.BwdURGFlags++
		}
	}
}

func (f *Flow) updateTCPFlags(tcp *layers.TCP) {
	if tcp.FIN {
		f.FINFlagCount++
	}
	if tcp.SYN {
		f.SYNFlagCount++
	}
	if tcp.RST {
		f.RSTFlagCount++
	}
	if tcp.PSH {
		f.PSHFlagCount++
	}
	if tcp.ACK {
		f.ACKFlagCount++
	}
	if tcp.URG {
		f.URGFlagCount++
	}
	if tcp.ECE {
		f.ECEFlagCount++
	}
	if tcp.CWR {
		f.CWEFlagCount++
	} // gopacket 中是 CWR
}

// GetMean 返回平均值
func GetMean(sum float64, count uint64) float64 {
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// GetStd 返回标准差
func GetStd(sum, sqSum float64, count uint64) float64 {
	if count <= 1 {
		return 0
	}
	mean := sum / float64(count)
	variance := (sqSum / float64(count)) - (mean * mean)
	if variance < 0 {
		variance = 0
	}
	return math.Sqrt(variance)
}
