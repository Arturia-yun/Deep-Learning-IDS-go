package server

import (
	"net/http"

	"go-ids/internal/loader"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ConfigUpdateRequest 用于接收前端更新阈值的 JSON 载荷
type ConfigUpdateRequest struct {
	Threshold float64 `json:"threshold" binding:"required,gt=0,lte=1"`
}

// GetEngineStatusHandler 获取当前的引擎及其配置状态
func GetEngineStatusHandler(c *gin.Context) {
	cfg := loader.GetConfig()
	if cfg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "系统配置尚未初始化"})
		return
	}

	// 抽出前端最关心的那部分属性
	c.JSON(http.StatusOK, gin.H{
		"model_path":           cfg.Detection.ModelPath,
		"scaler_path":          cfg.Detection.ScalerPath,
		"current_threshold":    cfg.Detection.Threshold,
		"suspicious_threshold": cfg.Detection.SuspiciousThreshold,
	})
}

// UpdateEngineConfigHandler 处理修改引擎参数的热请求
func UpdateEngineConfigHandler(c *gin.Context) {
	var req ConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置参数: " + err.Error()})
		return
	}

	// 调用底层热更新锁与落盘机制
	err := loader.UpdateDetectionThreshold(req.Threshold)
	if err != nil {
		logrus.Errorf("应用引擎新阈值失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "热更新失败: " + err.Error()})
		return
	}

	logrus.Infof("【热更新】管理员成功将检测阈值修改为: %.3f", req.Threshold)
	c.JSON(http.StatusOK, gin.H{
		"message":   "配置更新并持久化成功",
		"threshold": req.Threshold,
	})
}
