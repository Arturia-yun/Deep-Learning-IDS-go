package loader

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	GlobalConfig  *Config
	configMutex   sync.RWMutex
	globalCfgPath string
)

// Config 表示完整的配置结构
type Config struct {
	Capture     CaptureConfig     `yaml:"capture"`
	Networks    NetworksConfig    `yaml:"networks"`
	Flow        FlowConfig        `yaml:"flow"`
	Detection   DetectionConfig   `yaml:"detection"`
	Response    ResponseConfig    `yaml:"response"`
	Logging     LoggingConfig     `yaml:"logging"`
	Performance PerformanceConfig `yaml:"performance"`
}

// CaptureConfig 数据包捕获配置
type CaptureConfig struct {
	Interface   string `yaml:"interface"`
	Snaplen     int    `yaml:"snaplen"`
	Promiscuous bool   `yaml:"promiscuous"`
}

// NetworksConfig 网络定义配置
type NetworksConfig struct {
	HomeNet []string `yaml:"home_net"`
}

// FlowConfig 流管理配置
type FlowConfig struct {
	TCPTimeout      int `yaml:"tcp_timeout"`
	UDPTimeout      int `yaml:"udp_timeout"`
	MaxFlows        int `yaml:"max_flows"`
	CleanupInterval int `yaml:"cleanup_interval"`
}

// DetectionConfig 检测配置
type DetectionConfig struct {
	ModelPath            string  `yaml:"model_path"`
	ORTLibPath           string  `yaml:"ort_lib_path"` // ONNX Runtime 库路径
	ScalerPath           string  `yaml:"scaler_path"`
	LabelMapPath         string  `yaml:"label_map_path"`
	Threshold            float64 `yaml:"threshold"`
	SuspiciousThreshold  float64 `yaml:"suspicious_threshold"`
	SuspiciousCountLimit int     `yaml:"suspicious_count_limit"`
}

// ResponseConfig 响应配置
type ResponseConfig struct {
	EnableBlock   bool     `yaml:"enable_block"`
	BlockDuration int      `yaml:"block_duration"`
	Whitelist     []string `yaml:"whitelist"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	DecoderWorkers  int `yaml:"decoder_workers"`
	FeatureWorkers  int `yaml:"feature_workers"`
	PacketQueueSize int `yaml:"packet_queue_size"`
}

// Load 从YAML文件加载配置并初始化全局变量
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	configMutex.Lock()
	GlobalConfig = &config
	globalCfgPath = configPath
	configMutex.Unlock()

	return &config, nil
}

// GetConfig 安全地返回当前全局配置的副本（浅拷贝处理基础配置读取即可，因为只有Threshold动态修改）
func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return GlobalConfig
}

// UpdateDetectionThreshold 更新阈值并持久化
func UpdateDetectionThreshold(newThreshold float64) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	if GlobalConfig == nil {
		return fmt.Errorf("全局配置尚未初始化")
	}

	if newThreshold <= 0 || newThreshold > 1 {
		return fmt.Errorf("detection.threshold 必须在0-1之间")
	}

	// 更新内存
	GlobalConfig.Detection.Threshold = newThreshold

	// 序列化写回文件
	data, err := yaml.Marshal(GlobalConfig)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(globalCfgPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 验证捕获配置
	if c.Capture.Interface == "" {
		return fmt.Errorf("capture.interface 不能为空")
	}
	if c.Capture.Snaplen <= 0 {
		return fmt.Errorf("capture.snaplen 必须大于0")
	}

	// 验证流配置
	if c.Flow.TCPTimeout <= 0 {
		return fmt.Errorf("flow.tcp_timeout 必须大于0")
	}
	if c.Flow.UDPTimeout <= 0 {
		return fmt.Errorf("flow.udp_timeout 必须大于0")
	}
	if c.Flow.MaxFlows <= 0 {
		return fmt.Errorf("flow.max_flows 必须大于0")
	}

	// 验证检测配置
	if c.Detection.ModelPath == "" {
		return fmt.Errorf("detection.model_path 不能为空")
	}
	if c.Detection.ORTLibPath == "" {
		return fmt.Errorf("detection.ort_lib_path 不能为空")
	}
	if c.Detection.ScalerPath == "" {
		return fmt.Errorf("detection.scaler_path 不能为空")
	}
	if c.Detection.Threshold <= 0 || c.Detection.Threshold > 1 {
		return fmt.Errorf("detection.threshold 必须在0-1之间")
	}

	// 验证性能配置
	if c.Performance.DecoderWorkers <= 0 {
		return fmt.Errorf("performance.decoder_workers 必须大于0")
	}
	if c.Performance.FeatureWorkers <= 0 {
		return fmt.Errorf("performance.feature_workers 必须大于0")
	}

	return nil
}

// GetDefaultConfig 返回默认配置
func GetDefaultConfig() *Config {
	return &Config{
		Capture: CaptureConfig{
			Interface:   "eth0",
			Snaplen:     65535,
			Promiscuous: false,
		},
		Flow: FlowConfig{
			TCPTimeout:      60,
			UDPTimeout:      30,
			MaxFlows:        100000,
			CleanupInterval: 10,
		},
		Detection: DetectionConfig{
			ModelPath:            "config/model.onnx",
			ORTLibPath:           "onnxruntime.dll", // 默认假设在当前目录或系统路径
			ScalerPath:           "config/scaler_params.json",
			LabelMapPath:         "config/label_map.json",
			Threshold:            0.8,
			SuspiciousThreshold:  0.6,
			SuspiciousCountLimit: 3,
		},
		Response: ResponseConfig{
			EnableBlock:   true,
			BlockDuration: 3600,
			Whitelist:     []string{},
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			FilePath:   "logs/ids.log",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
		},
		Performance: PerformanceConfig{
			DecoderWorkers:  4,
			FeatureWorkers:  2,
			PacketQueueSize: 10000,
		},
	}
}
