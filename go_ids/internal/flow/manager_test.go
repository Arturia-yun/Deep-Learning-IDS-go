package flow

import (
	"net"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Helper to create a dummy packet
func createKeyAndPacket(t *testing.T, srcIP, dstIP string, srcPort, dstPort uint16) (FlowKey, gopacket.Packet) {
	sIP := net.ParseIP(srcIP)
	dIP := net.ParseIP(dstIP)

	key := NewFlowKey(sIP, dIP, srcPort, dstPort, layers.IPProtocolTCP)

	// Create minimal packet for Flow updates (needs metadata timestamp and length)
	// We can leave layers empty for simple Flow Manager tests, or fill them if Flow.Update reads them.
	// Flow.Update reads IP/TCP layers, so we should populate them.

	eth := layers.Ethernet{EthernetType: layers.EthernetTypeIPv4}
	ip := layers.IPv4{
		SrcIP:    sIP,
		DstIP:    dIP,
		Protocol: layers.IPProtocolTCP,
		Version:  4,
		IHL:      5,
	}
	tcp := layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(dstPort),
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true}
	gopacket.SerializeLayers(buffer, opts, &eth, &ip, &tcp)
	pkt := gopacket.NewPacket(buffer.Bytes(), layers.LayerTypeEthernet, gopacket.Default)

	return key, pkt
}

func TestManager_GetOrCreate(t *testing.T) {
	mgr := NewManager(time.Minute)

	// 1. Create Forward Flow
	key1, pkt1 := createKeyAndPacket(t, "192.168.1.1", "192.168.1.2", 12345, 80)
	flow1, isFwd1 := mgr.GetOrCreate(key1, pkt1)

	if flow1 == nil {
		t.Fatal("Failed to create flow")
	}
	if !isFwd1 {
		t.Error("Expected forward direction for new flow")
	}
	if mgr.Count() != 1 {
		t.Errorf("Expected 1 flow, got %d", mgr.Count())
	}

	// 2. Access same flow (Forward)
	flow2, isFwd2 := mgr.GetOrCreate(key1, pkt1)
	if flow2 != flow1 {
		t.Error("GetOrCreate returned different flow object for same key")
	}
	if !isFwd2 {
		t.Error("Expected forward direction for existing flow")
	}

	// 3. Access same flow (Backward)
	key2, pkt2 := createKeyAndPacket(t, "192.168.1.2", "192.168.1.1", 80, 12345)

	// Verify key2 is reverse of key1
	if key1.Reverse() != key2 {
		t.Fatal("Key generation failed to produce reverse key")
	}

	flow3, isFwd3 := mgr.GetOrCreate(key2, pkt2)
	if flow3 != flow1 {
		t.Error("GetOrCreate returned different flow object for reverse key")
	}
	if isFwd3 {
		t.Error("Expected backward direction for reverse key")
	}
	if mgr.Count() != 1 {
		t.Errorf("Expected 1 flow, got %d", mgr.Count())
	}
}

func TestManager_Cleanup(t *testing.T) {
	timeout := 100 * time.Millisecond
	mgr := NewManager(timeout)

	key, pkt := createKeyAndPacket(t, "10.0.0.1", "10.0.0.2", 1000, 2000)
	f, _ := mgr.GetOrCreate(key, pkt)

	// Set LastTime to past
	f.LastTime = time.Now().Add(-200 * time.Millisecond)

	// Cleanup
	expired := mgr.Cleanup()

	if len(expired) != 1 {
		t.Errorf("Expected 1 expired flow, got %d", len(expired))
	}
	if mgr.Count() != 0 {
		t.Errorf("Expected 0 active flows, got %d", mgr.Count())
	}
}
