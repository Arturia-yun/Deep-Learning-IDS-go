package feature

import (
	"fmt"
	"testing"
)

func TestScalerLoad(t *testing.T) {
	// 这是一个集成测试，依赖于 model_training/dataset/scaler_params.json
	scaler, err := NewScaler("../../../model_training/dataset/scaler_params.json")
	if err != nil {
		t.Fatalf("加载标准化参数失败: %v", err)
	}

	fmt.Printf("成功加载了 %d 个特征的参数\n", len(scaler.GetFeatureNames()))

	if len(scaler.GetFeatureNames()) != 78 {
		t.Errorf("特征数量不匹配: 期望 78, 得到 %d", len(scaler.GetFeatureNames()))
	}
}

func TestTransform(t *testing.T) {
	scaler, err := NewScaler("../../../model_training/dataset/scaler_params.json")
	if err != nil {
		t.Skip("跳过测试：找不到参数文件")
	}

	// 创建一个全 0 的特征向量
	features := make([]float32, 78)
	scaled, err := scaler.Transform(features)
	if err != nil {
		t.Fatalf("标准化失败: %v", err)
	}

	if len(scaled) != 78 {
		t.Errorf("标准化后的维度不匹配")
	}

	fmt.Printf("全0向量标准化后的前5个值: %v\n", scaled[:5])
}
