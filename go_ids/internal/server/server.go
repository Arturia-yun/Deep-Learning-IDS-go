package server

import (
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// StartTime tracks when the IDS system started
var StartTime time.Time

// FlowCounter Interface to avoid strict dependency coupling if needed,
// though we can import standard flow package if no cycle.
// FlowBrief contains summary info for dashboard validation
type FlowBrief struct {
	SrcPort  uint16 `json:"src_port"`
	DstPort  uint16 `json:"dst_port"`
	Protocol string `json:"protocol"` // "TCP" or "UDP"
	Duration string `json:"duration"` // e.g. "12s"
}

// FlowCounter Interface to avoid strict dependency coupling if needed,
// though we can import standard flow package if no cycle.
type FlowCounter interface {
	Count() int
	GetRecentFlows(limit int) []FlowBrief
}

var flowCounter FlowCounter

// SetFlowCounter allows main to inject the flow manager
func SetFlowCounter(fc FlowCounter) {
	flowCounter = fc
}

// TrafficTracker manages real-time bandwidth statistics
type TrafficTracker struct {
	BytesIn  uint64  // Rx (Download)
	BytesOut uint64  // Tx (Upload)
	RateIn   float64 // Mbps
	RateOut  float64 // Mbps
	mu       sync.RWMutex
}

var stats TrafficTracker

// AddTraffic increments traffic counters
func AddTraffic(in int, out int) {
	stats.mu.Lock()
	stats.BytesIn += uint64(in)
	stats.BytesOut += uint64(out)
	stats.mu.Unlock()
}

// GetRates returns the current traffic rates (in, out) in Mbps
func GetRates() (float64, float64) {
	stats.mu.RLock()
	defer stats.mu.RUnlock()
	return stats.RateIn, stats.RateOut
}

// StartServer initializes and runs the Gin HTTP server
func StartServer(port string) error {
	StartTime = time.Now()

	// Start Traffic Monitor Ticker (1s interval)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		var lastIn, lastOut uint64

		for range ticker.C {
			stats.mu.Lock()
			currIn := stats.BytesIn
			currOut := stats.BytesOut

			// Calculate bits per second: (diff bytes * 8) / 1s / 1M
			diffIn := currIn - lastIn
			diffOut := currOut - lastOut

			stats.RateIn = float64(diffIn*8) / 1000000.0
			stats.RateOut = float64(diffOut*8) / 1000000.0

			lastIn = currIn
			lastOut = currOut
			stats.mu.Unlock()
		}
	}()

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	// Custom logger middleware could be added here

	// CORS Configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allow all for dev, restrict in prod
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API Routes
	api := r.Group("/api")
	{
		api.GET("/events", ServerSentEventsHandler)
		api.GET("/alerts", GetAlertsHandler)
		api.GET("/status", SystemStatusHandler)
		api.GET("/stats/threats", GetThreatStatsHandler)

		// AI 引擎管理路由
		api.GET("/engine/status", GetEngineStatusHandler)
		api.POST("/engine/config", UpdateEngineConfigHandler)
	}

	// Start SSE Manager
	go Manager.Listen()

	// Run Server
	return r.Run(":" + port)
}
