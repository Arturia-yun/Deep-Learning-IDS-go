package inference

import (
	"fmt"
	"math"
	"path/filepath"

	ort "github.com/yalue/onnxruntime_go"
)

// Prediction 存储推理结果
type Prediction struct {
	Label       string
	Probability float32
	LabelIndex  int
}

// Engine 封装了 ONNX 推理逻辑
type Engine struct {
	session *ort.DynamicAdvancedSession
	labels  []string
}

// NewEngine 初始化推理引擎
// modelPath: ONNX 模型文件路径
// ortLibPath: onnxruntime.dll (Windows) 或 .so (Linux) 的路径
func NewEngine(modelPath string, ortLibPath string) (*Engine, error) {
	// 1. 设置 ONNX Runtime 库路径
	// 注意：在整个进程中只需要设置一次
	if !ort.IsInitialized() {
		absLibPath, err := filepath.Abs(ortLibPath)
		if err != nil {
			return nil, fmt.Errorf("无法解析库路径: %v", err)
		}
		ort.SetSharedLibraryPath(absLibPath)
		err = ort.InitializeEnvironment()
		if err != nil {
			return nil, fmt.Errorf("初始化 ONNX Runtime 环境失败: %v", err)
		}
	}

	// 2. 标签映射 (需与 Python 端 label_map.json 一致)
	labels := []string{"Benign", "Bot", "Brute Force", "DoS", "PortScan", "Web Attack"}

	// 3. 创建推理会话
	// 根据报错，NewDynamicAdvancedSession 期望 (modelPath, inputNames, outputNames, options)
	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		[]string{"features"}, // 输入节点名
		[]string{"logits"},   // 输出节点名
		nil,                  // SessionOptions
	)
	if err != nil {
		return nil, fmt.Errorf("创建 ONNX 会话失败: %v", err)
	}

	return &Engine{
		session: session,
		labels:  labels,
	}, nil
}

// Predict 执行推理
func (e *Engine) Predict(features []float32) (Prediction, error) {
	if len(features) != 78 {
		return Prediction{}, fmt.Errorf("输入特征维度错误: 期望 78, 得到 %d", len(features))
	}

	// 准备输入 Tensor
	inputTensor, err := ort.NewTensor(ort.NewShape(1, 78), features)
	if err != nil {
		return Prediction{}, fmt.Errorf("创建输入 Tensor 失败: %v", err)
	}
	defer inputTensor.Destroy()

	// 准备输出 Tensor (用于接收结果)
	outputTensor, err := ort.NewEmptyTensor[float32](ort.NewShape(1, 6))
	if err != nil {
		return Prediction{}, fmt.Errorf("创建输出 Tensor 失败: %v", err)
	}
	defer outputTensor.Destroy()

	// 执行推理
	err = e.session.Run(
		[]ort.ArbitraryTensor{inputTensor},
		[]ort.ArbitraryTensor{outputTensor},
	)
	if err != nil {
		return Prediction{}, fmt.Errorf("推理执行失败: %v", err)
	}

	// 获取输出数据
	outputData := outputTensor.GetData()

	// 应用 Softmax 将 Logits 转换为 [0, 1] 之间的概率
	maxLogit := outputData[0]
	for _, l := range outputData {
		if l > maxLogit {
			maxLogit = l
		}
	}

	var sum float32
	probs := make([]float32, len(outputData))
	for i, l := range outputData {
		// 为了防止指数爆炸，减去 maxLogit
		probs[i] = float32(math.Exp(float64(l - maxLogit)))
		sum += probs[i]
	}

	// 寻找概率最大的类别
	maxIdx := 0
	maxProb := float32(-1.0)
	for i := range probs {
		probs[i] /= sum
		if probs[i] > maxProb {
			maxProb = probs[i]
			maxIdx = i
		}
	}

	return Prediction{
		Label:       e.labels[maxIdx],
		Probability: maxProb,
		LabelIndex:  maxIdx,
	}, nil
}

// Close 释放资源
func (e *Engine) Close() {
	if e.session != nil {
		e.session.Destroy()
	}
	// 注意：通常不建议在 Engine 关闭时调用 ort.Destroy()，
	// 因为其他 Engine 实例可能还在使用。
}
