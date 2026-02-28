package decoder

import (
	"net"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func createTestPacket(t *testing.T, payload []byte) gopacket.Packet {
	// Construct a packet
	eth := layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := layers.IPv4{
		SrcIP:    net.IP{192, 168, 1, 1},
		DstIP:    net.IP{192, 168, 1, 2},
		Protocol: layers.IPProtocolTCP,
		Version:  4,
		TTL:      64,
	}
	tcp := layers.TCP{
		SrcPort: layers.TCPPort(12345),
		DstPort: layers.TCPPort(80),
		SYN:     true,
		Window:  65535,
	}
	tcp.SetNetworkLayerForChecksum(&ip)

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, &eth, &ip, &tcp, gopacket.Payload(payload))
	if err != nil {
		t.Fatalf("Failed to serialize packet: %v", err)
	}

	return gopacket.NewPacket(buffer.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
}

func TestDecodeTCP(t *testing.T) {
	d := NewDecoder()
	payload := []byte("hello")
	pkt := createTestPacket(t, payload)

	decoded, err := d.Decode(pkt)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded == nil {
		t.Fatal("Decoded packet is nil")
	}

	if decoded.SrcIP != "192.168.1.1" {
		t.Errorf("Expected SrcIP 192.168.1.1, got %s", decoded.SrcIP)
	}
	if decoded.DstPort != 80 {
		t.Errorf("Expected DstPort 80, got %d", decoded.DstPort)
	}
	if decoded.Protocol != 6 { // TCP
		t.Errorf("Expected Protocol 6, got %d", decoded.Protocol)
	}
	if !decoded.TCPFlags.SYN {
		t.Error("Expected SYN flag to be true")
	}
}

func TestDecodeNonIP(t *testing.T) {
	d := NewDecoder()
	// Create ARP packet (Non-IP)
	eth := layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       net.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		EthernetType: layers.EthernetTypeARP,
	}
	arp := layers.ARP{
		AddrType: layers.LinkTypeEthernet,
		Protocol: layers.EthernetTypeIPv4,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	gopacket.SerializeLayers(buffer, opts, &eth, &arp)
	pkt := gopacket.NewPacket(buffer.Bytes(), layers.LayerTypeEthernet, gopacket.Default)

	decoded, err := d.Decode(pkt)
	if err != nil {
		t.Errorf("Decode returned error for Non-IP packet: %v", err)
	}
	if decoded != nil {
		t.Error("Expected nil for Non-IP packet, got struct")
	}
}
