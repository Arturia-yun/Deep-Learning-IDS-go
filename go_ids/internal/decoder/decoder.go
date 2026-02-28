package decoder

import (
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// DecodedPacket 包含了从原始包中提取的结构化信息
type DecodedPacket struct {
	Timestamp time.Time
	SrcIP     string
	DstIP     string
	SrcPort   uint16
	DstPort   uint16
	Protocol  uint8 // 6 为 TCP, 17 为 UDP
	Payload   []byte
	Length    int

	// TCP 特有字段
	TCPFlags layers.TCP
	Window   uint16

	// IP 特有字段
	TTL uint8
}

// Decoder 定义了解码接口
type Decoder struct {
	eth  layers.Ethernet
	ip4  layers.IPv4
	ip6  layers.IPv6
	tcp  layers.TCP
	udp  layers.UDP
	icmp layers.ICMPv4

	parser *gopacket.DecodingLayerParser
}

// NewDecoder 创建一个新的解码器
func NewDecoder() *Decoder {
	d := &Decoder{}
	d.parser = gopacket.NewDecodingLayerParser(
		layers.LayerTypeEthernet,
		&d.eth,
		&d.ip4,
		&d.ip6,
		&d.tcp,
		&d.udp,
		&d.icmp,
	)
	// 忽略未知层
	d.parser.IgnoreUnsupported = true
	return d
}

// Decode 解析一个原始数据包
func (d *Decoder) Decode(packet gopacket.Packet) (*DecodedPacket, error) {
	decoded := &DecodedPacket{
		Timestamp: packet.Metadata().Timestamp,
		Length:    packet.Metadata().Length,
	}

	// 使用 gopacket 的标准解析方式
	// 或者使用 DecodingLayerParser 性能更高，但这里先用标准方式简化实现

	// 1. 获取网络层 (IP)
	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		decoded.SrcIP = ip.SrcIP.String()
		decoded.DstIP = ip.DstIP.String()
		decoded.Protocol = uint8(ip.Protocol)
		decoded.TTL = ip.TTL
	} else if ipLayer := packet.Layer(layers.LayerTypeIPv6); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv6)
		decoded.SrcIP = ip.SrcIP.String()
		decoded.DstIP = ip.DstIP.String()
		decoded.Protocol = uint8(ip.NextHeader)
		decoded.TTL = ip.HopLimit
	} else {
		return nil, nil // 非 IP 包，忽略
	}

	// 2. 获取传输层 (TCP/UDP)
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		decoded.SrcPort = uint16(tcp.SrcPort)
		decoded.DstPort = uint16(tcp.DstPort)
		decoded.TCPFlags = *tcp
		decoded.Window = tcp.Window
		decoded.Payload = tcp.Payload
	} else if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		decoded.SrcPort = uint16(udp.SrcPort)
		decoded.DstPort = uint16(udp.DstPort)
		decoded.Payload = udp.Payload
	}

	return decoded, nil
}
