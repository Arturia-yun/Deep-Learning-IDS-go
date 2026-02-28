package feature

import (
	"encoding/json"
	"fmt"
	"os"
)

// ScalerParams 存储从 JSON 加载的标准化参数
type ScalerParams struct {
	Mean         []float64 `json:"mean"`
	Scale        []float64 `json:"scale"`
	FeatureNames []string  `json:"feature_names"`
}

// Scaler 负责特征的标准化处理
type Scaler struct {
	params ScalerParams
}

// NewScaler 从指定文件加载参数并创建 Scaler
func NewScaler(filePath string) (*Scaler, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法读取标准化参数文件: %v", err)
	}

	var params ScalerParams
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("解析标准化参数失败: %v", err)
	}

	return &Scaler{params: params}, nil
}

// Transform 对特征向量进行标准化: (x - mean) / scale
func (s *Scaler) Transform(features []float32) ([]float32, error) {
	if len(features) != len(s.params.Mean) {
		return nil, fmt.Errorf("特征维度不匹配: 期望 %d, 得到 %d", len(s.params.Mean), len(features))
	}

	scaled := make([]float32, len(features))
	for i := range features {
		// 避免除以 0
		scale := s.params.Scale[i]
		if scale == 0 {
			scale = 1.0
		}
		scaled[i] = float32((float64(features[i]) - s.params.Mean[i]) / scale)
	}

	return scaled, nil
}

// GetFeatureNames 返回特征名称列表
func (s *Scaler) GetFeatureNames() []string {
	return s.params.FeatureNames
}
