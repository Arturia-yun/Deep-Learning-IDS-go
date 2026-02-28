package capture

import (
	"fmt"
	"log"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// PcapSource 是基于 pcap 库的实现
type PcapSource struct {
	handle *pcap.Handle
	source *gopacket.PacketSource
}

// NewPcapSource 创建一个新的实时抓包源
func NewPcapSource(device string, snaplen int32, promisc bool) (*PcapSource, error) {
	// 打开设备进行实时抓包
	handle, err := pcap.OpenLive(device, snaplen, promisc, pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("无法打开设备 %s: %v", device, err)
	}

	source := gopacket.NewPacketSource(handle, handle.LinkType())
	
	return &PcapSource{
		handle: handle,
		source: source,
	}, nil
}

// NewFileSource 从 pcap 文件读取数据包
func NewFileSource(filename string) (*PcapSource, error) {
	handle, err := pcap.OpenOffline(filename)
	if err != nil {
		return nil, fmt.Errorf("无法打开 pcap 文件 %s: %v", filename, err)
	}

	source := gopacket.NewPacketSource(handle, handle.LinkType())

	return &PcapSource{
		handle: handle,
		source: source,
	}, nil
}

// Packets 返回数据包通道
func (p *PcapSource) Packets() <-chan gopacket.Packet {
	return p.source.Packets()
}

// Close 关闭抓包源
func (p *PcapSource) Close() {
	if p.handle != nil {
		p.handle.Close()
	}
}

// GetStats 获取抓包统计信息
func (p *PcapSource) GetStats() (Stats, error) {
	s, err := p.handle.Stats()
	if err != nil {
		return Stats{}, err
	}
	return Stats{
		PacketsReceived:  int64(s.PacketsReceived),
		PacketsDropped:   int64(s.PacketsDropped),
		PacketsIfDropped: int64(s.PacketsIfDropped),
	}, nil
}

// ListDevices 列出所有可用的网络接口
func ListDevices() {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("可用网络接口:")
	for _, device := range devices {
		fmt.Printf("名称: %s\n", device.Name)
		fmt.Printf("描述: %s\n", device.Description)
		for _, address := range device.Addresses {
			fmt.Printf("- IP 地址: %s\n", address.IP)
		}
		fmt.Println("-----------------------------------")
	}
}

