package capture

import (
	"github.com/google/gopacket"
)

// PacketSource 定义了数据包获取的通用接口
type PacketSource interface {
	// Packets 返回一个用于接收数据包的通道
	Packets() <-chan gopacket.Packet
	// Close 关闭抓包源
	Close()
}

// Stats 提供抓包统计信息
type Stats struct {
	PacketsReceived  int64
	PacketsDropped   int64
	PacketsIfDropped int64
}
