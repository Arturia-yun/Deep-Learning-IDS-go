package flow

import (
	"sync"
	"time"

	"go-ids/internal/server"

	"github.com/google/gopacket"
)

// Manager 管理系统中所有的活跃流
type Manager struct {
	flows   map[FlowKey]*Flow
	mu      sync.RWMutex
	timeout time.Duration
}

// NewManager 创建一个新的流管理器
func NewManager(timeout time.Duration) *Manager {
	return &Manager{
		flows:   make(map[FlowKey]*Flow),
		timeout: timeout,
	}
}

// GetOrCreate 获取现有流或创建一个新流
// 它会自动识别方向：如果找到 Key 或其 Reverse Key，则返回该流并告知方向
func (m *Manager) GetOrCreate(key FlowKey, pkt gopacket.Packet) (*Flow, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 尝试直接匹配（正向）
	if f, ok := m.flows[key]; ok {
		return f, true
	}

	// 2. 尝试匹配反向键（反向）
	revKey := key.Reverse()
	if f, ok := m.flows[revKey]; ok {
		return f, false
	}

	// 3. 都不存在，创建新流（默认为正向）
	f := NewFlow(key, pkt)
	m.flows[key] = f
	return f, true
}

// Cleanup 清理超时的流
// 返回被清理掉的流列表，以便进行最后的特征提取和推理
func (m *Manager) Cleanup() []*Flow {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var expired []*Flow

	for key, f := range m.flows {
		if now.Sub(f.LastTime) > m.timeout {
			expired = append(expired, f)
			delete(m.flows, key)
		}
	}

	return expired
}

// Count 返回当前管理的流数量
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.flows)
}

// GetRecentFlows returns a list of active flows for the dashboard
// Note: Return type matches server.FlowBrief, but we need to import server to use it explicitly
// Or we can return []interface{} and cast.
// Ideally, flow shouldn't import server (downwards).
// Currently main.go injects flow -> server.
// So flow can import server.
// However, circular dependency risk if server imports flow. (Server doesn't).
// We will add import "go-ids/internal/server" to this file.
func (m *Manager) GetRecentFlows(limit int) []server.FlowBrief {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var flows []server.FlowBrief
	count := 0
	now := time.Now()

	for _, f := range m.flows {
		if count >= limit {
			break
		}

		dur := now.Sub(f.StartTime).Truncate(time.Second).String()

		proto := "TCP"
		if f.Key.Proto == 17 {
			proto = "UDP"
		}

		flows = append(flows, server.FlowBrief{
			SrcPort:  f.Key.SrcPort,
			DstPort:  f.Key.DstPort,
			Protocol: proto,
			Duration: dur,
		})
		count++
	}
	return flows
}
