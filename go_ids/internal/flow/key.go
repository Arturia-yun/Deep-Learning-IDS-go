package flow

import (
	"fmt"
	"net"

	"github.com/google/gopacket/layers"
)

// FlowKey 定义了一个流的唯一标识（五元组）
type FlowKey struct {
	SrcIP   string
	DstIP   string
	SrcPort uint16
	DstPort uint16
	Proto   layers.IPProtocol
}

// String 返回流键的字符串表示，方便日志记录
func (k FlowKey) String() string {
	return fmt.Sprintf("%s:%d -> %s:%d [%s]", k.SrcIP, k.SrcPort, k.DstIP, k.DstPort, k.Proto)
}

// NewFlowKey 从 IP 和传输层创建一个流键
// 注意：为了支持双向流识别，有些系统会对 IP/Port 进行排序。
// 但在 CIC-IDS2017 中，我们需要区分 Forward（源到目的）和 Backward（目的到源）。
// 因此我们保留原始方向，在 FlowManager 中处理双向匹配。
func NewFlowKey(srcIP, dstIP net.IP, srcPort, dstPort uint16, proto layers.IPProtocol) FlowKey {
	return FlowKey{
		SrcIP:   srcIP.String(),
		DstIP:   dstIP.String(),
		SrcPort: srcPort,
		DstPort: dstPort,
		Proto:   proto,
	}
}

// Reverse 返回该流键的反向流键
func (k FlowKey) Reverse() FlowKey {
	return FlowKey{
		SrcIP:   k.DstIP,
		DstIP:   k.SrcIP,
		SrcPort: k.DstPort,
		DstPort: k.SrcPort,
		Proto:   k.Proto,
	}
}
