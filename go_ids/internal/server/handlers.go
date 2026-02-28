package server

import (
	"net/http"
	"strconv"
	"time"

	"go-ids/internal/db"

	"github.com/gin-gonic/gin"
)

// GetAlertsHandler retrieves historical alerts
func GetAlertsHandler(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	alerts, err := db.GetRecentAlerts(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

// SystemStatusHandler returns simple system stats
func SystemStatusHandler(c *gin.Context) {
	// TODO: Integrate actual stats from FlowManager for active flows
	uptime := time.Since(StartTime)

	in, out := GetRates()
	activeFlows := 0
	var flowList []FlowBrief
	if flowCounter != nil {
		activeFlows = flowCounter.Count()
		flowList = flowCounter.GetRecentFlows(20) // Top 20
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "running",
		"server_time":  time.Now(),
		"start_time":   StartTime,
		"uptime_str":   uptime.String(),
		"uptime_sec":   uptime.Seconds(),
		"traffic_in":   in,  // Mbps
		"traffic_out":  out, // Mbps
		"active_flows": activeFlows,
		"flow_list":    flowList,
	})
}

// GetThreatStatsHandler handles the chart data request
func GetThreatStatsHandler(c *gin.Context) {
	rangeType := c.DefaultQuery("range", "Day") // Day, Week, Month

	points, err := db.GetThreatStats(rangeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, points)
}
