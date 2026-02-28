package feature

import (
	"net"
	"testing"
	"time"

	"go-ids/internal/flow"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func createTestFlow(t *testing.T) *flow.Flow {
	srcIP := net.IP{192, 168, 1, 1}
	dstIP := net.IP{10, 0, 0, 1}
	key := flow.NewFlowKey(srcIP, dstIP, 12345, 80, layers.IPProtocolTCP)

	eth := layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := layers.IPv4{
		SrcIP: srcIP, DstIP: dstIP,
		Protocol: layers.IPProtocolTCP,
		Version:  4, IHL: 5, Length: 60,
	}
	tcp := layers.TCP{SrcPort: 12345, DstPort: 80, Window: 1000, SYN: true}
	tcp.SetNetworkLayerForChecksum(&ip)

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	err := gopacket.SerializeLayers(buffer, opts, &eth, &ip, &tcp)
	if err != nil {
		t.Fatalf("SerializeLayers failed: %v", err)
	}
	// Set timestamp directly on metadata
	pkt := gopacket.NewPacket(buffer.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
	pkt.Metadata().Timestamp = time.Now()
	pkt.Metadata().Length = 60
	pkt.Metadata().CaptureInfo.CaptureLength = 60
	pkt.Metadata().CaptureInfo.Length = 60

	if tcpLayer := pkt.Layer(layers.LayerTypeTCP); tcpLayer == nil {
		t.Fatal("Packet created without TCP layer")
	}

	f := flow.NewFlow(key, pkt)
	f.Update(pkt, true) // Forward packet (SYN)

	return f
}

func TestExtractor_Extract(t *testing.T) {
	e := NewExtractor()
	f := createTestFlow(t)

	// Add a backward packet
	// ... (omitted for brevity, relying on the single forward packet init)

	features := e.Extract(f)

	if len(features) != 78 {
		t.Errorf("Expected 78 features, got %d", len(features))
	}

	// Check DstPort (Index 0)
	if features[0] != 80 {
		t.Errorf("Expected DstPort 80, got %f", features[0])
	}

	// Check Total Fwd Packets (Index 2) - NewFlow doesn't increment count, Update does.
	// NewFlow initializes. Update(pkt, true) increments. So count should be 1?
	// Let's check logic in flow.go. NewFlow sets stats? No, NewFlow just inits struct.
	// The Update call in createTestFlow handles the stats.

	// In createTestFlow we call f.Update(pkt, true). So FwdPackets should be 1.
	if features[2] != 1 {
		t.Errorf("Expected 1 Fwd Packet, got %f", features[2])
	}

	// Check SYN Flag Count (Index 44)
	if features[44] != 1 {
		t.Errorf("Expected 1 SYN Flag, got %f", features[44])
	}
}
