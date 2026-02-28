package loader

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// 测试加载配置文件
	config, err := Load("../../config/config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置值
	if config.Capture.Interface == "" {
		t.Error("capture.interface 不能为空")
	}
	if config.Detection.Threshold <= 0 {
		t.Error("detection.threshold 必须大于0")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	// 验证默认值
	if config.Capture.Interface != "eth0" {
		t.Errorf("期望默认interface为eth0，实际为: %s", config.Capture.Interface)
	}
	if config.Flow.TCPTimeout != 60 {
		t.Errorf("期望默认TCP超时为60，实际为: %d", config.Flow.TCPTimeout)
	}
}

func TestValidateConfig(t *testing.T) {
	config := GetDefaultConfig()

	// 正常配置应该通过验证
	if err := config.Validate(); err != nil {
		t.Errorf("默认配置验证失败: %v", err)
	}

	// 测试无效配置
	config.Capture.Interface = ""
	if err := config.Validate(); err == nil {
		t.Error("空interface应该验证失败")
	}

	config = GetDefaultConfig()
	config.Detection.Threshold = 1.5 // 无效阈值
	if err := config.Validate(); err == nil {
		t.Error("无效阈值应该验证失败")
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	if err == nil {
		t.Error("加载不存在的文件应该返回错误")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	// 创建临时无效YAML文件
	tmpFile := "test_invalid.yaml"
	os.WriteFile(tmpFile, []byte("invalid: yaml: content: [unclosed"), 0644)
	defer os.Remove(tmpFile)

	_, err := Load(tmpFile)
	if err == nil {
		t.Error("加载无效YAML应该返回错误")
	}
}
