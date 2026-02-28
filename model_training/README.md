# 模型训练模块

基于深度学习的入侵检测系统 - Python训练端

本项目实现了使用PyTorch训练入侵检测模型，并导出为ONNX格式供Go程序部署使用。

## 目录结构

```
model_training/
├── dataset/                    # 数据集目录
│   ├── MachineLearningCVE/    # 原始数据集（CIC-IDS2017）
│   ├── cicids2017_dev.csv     # 开发集（10%采样，约28万条）
│   ├── train.csv              # 训练集（70%，约19.8万条）
│   ├── val.csv                # 验证集（12.5%，约4.2万条）
│   ├── test.csv               # 测试集（12.5%，约4.2万条）
│   ├── label_map.json         # 标签映射（6类攻击类型）
│   └── scaler_params.json     # 标准化参数（Go程序必需）
│
├── models/                     # 训练好的模型（输出目录）
│   ├── checkpoints/            # PyTorch模型检查点
│   │   └── best_model.pth     # 最佳模型检查点
│   ├── onnx/                   # ONNX模型文件
│   │   └── ids_model.onnx     # ONNX模型（Go程序使用）
│   └── logs/                   # 训练日志
│       └── training_history.json  # 训练历史记录
│
├── src/                        # 源代码
│   ├── data/                   # 数据处理模块
│   │   ├── __init__.py
│   │   ├── process_dataset.py  # 数据整合与采样
│   │   └── preprocessing.py    # 数据预处理与标准化
│   │
│   └── models/                  # 模型训练模块
│       ├── __init__.py
│       ├── model.py            # 模型定义（MLP）
│       ├── train.py            # 训练脚本
│       └── export_onnx.py      # ONNX导出脚本
│
├── requirements.txt            # Python依赖包
└── README.md                   # 本文档
```

## 使用流程

### 1. 数据处理

#### 步骤1: 数据整合与采样
```bash
python model_training/src/data/process_dataset.py
```
- 读取原始CSV文件并合并
- 进行标签映射和清洗
- 生成开发集（10%采样）和标签映射

#### 步骤2: 数据预处理
```bash
python model_training/src/data/preprocessing.py
```
- 数据清洗（处理Inf/NaN）
- 标签编码
- 数据集划分（75%训练 / 12.5%验证 / 12.5%测试）
- 特征标准化
- 生成训练集、验证集、测试集和标准化参数

### 2. 模型训练

#### 步骤3: 训练模型
```bash
python model_training/src/models/train.py
```
- 加载训练集和验证集
- 训练神经网络模型
- 保存最佳模型检查点
- 输出训练历史

#### 步骤4: 导出ONNX模型
```bash
python model_training/src/models/export_onnx.py
```
- 加载训练好的模型
- 导出为ONNX格式
- 验证ONNX模型与PyTorch模型一致性

## 模型结构

### 网络架构
- **类型**: 多层感知机（MLP）
- **输入维度**: 78个特征（CIC-IDS2017全部特征）
- **输出维度**: 6个类别（Benign, Bot, Brute Force, DoS, PortScan, Web Attack）
- **隐藏层**: [128, 64, 32]
- **激活函数**: ReLU
- **Dropout**: 0.3（防止过拟合）

### 训练配置
- **批次大小**: 256
- **学习率**: 0.001
- **优化器**: Adam
- **学习率调度**: ReduceLROnPlateau（当验证损失不下降时自动降低学习率）
- **早停机制**: 耐心值=10（验证集F1连续10轮不提升则停止训练）
- **最大训练轮数**: 50

## 模型性能

### 最终验证集性能
- **准确率 (Accuracy)**: 97.98%
- **精确率 (Precision)**: 97.94%
- **召回率 (Recall)**: 97.98%
- **F1分数**: 97.93%

### 训练过程
- **训练轮数**: 29轮（早停触发）
- **最佳模型**: 第19轮（F1=97.93%）
- **训练集准确率**: 97.67%
- **验证集准确率**: 97.63%
- **过拟合情况**: 无（训练集与验证集性能接近）

## 输出文件

训练完成后，`models/` 目录将包含：
- `checkpoints/best_model.pth`: PyTorch模型检查点（包含模型权重、优化器状态等）
- `onnx/ids_model.onnx`: ONNX模型文件（供Go程序使用）✨
- `logs/training_history.json`: 训练历史记录（每轮的损失和指标）

**目录说明：**
- `checkpoints/`: 存放PyTorch模型检查点文件
- `onnx/`: 存放导出的ONNX模型文件（Go程序部署必需）
- `logs/`: 存放训练日志和历史记录

## 部署到Go程序

### 必需文件
1. **`models/onnx/ids_model.onnx`** - ONNX模型文件
2. **`dataset/scaler_params.json`** - 标准化参数（均值和标准差）
3. **`dataset/label_map.json`** - 标签映射（可选，用于输出可读的类别名称）

### 文件位置
将这些文件复制到Go项目的相应目录：
```
go_ids/
├── model.onnx              # 从 models/onnx/ids_model.onnx 复制
└── config/
    └── scaler_params.json  # 从 dataset/scaler_params.json 复制
```

## 注意事项

### 依赖安装
1. 确保已安装所有依赖：`pip install -r requirements.txt`
2. PyTorch安装可能需要较长时间（约600MB）

### 执行顺序
1. **数据处理步骤必须按顺序执行**：
   - 先运行 `process_dataset.py` 生成开发集
   - 再运行 `preprocessing.py` 生成训练集
2. **ONNX模型导出前必须先完成模型训练**

### 重要文件
1. **`scaler_params.json`** 文件对Go程序部署至关重要，必须与训练时使用的参数一致
2. **`label_map.json`** 用于将模型输出的数字标签转换为可读的攻击类型名称

### 性能优化建议
- 如果训练时间过长，可以调整 `train.py` 中的 `BATCH_SIZE` 和 `NUM_EPOCHS`
- 如果内存不足，可以减少 `process_dataset.py` 中的 `SAMPLE_RATE`（当前为0.1，即10%）

## 常见问题

### Q: 训练过程中出现CUDA错误？
A: 代码会自动检测并使用CPU，如果系统没有GPU，会自动使用CPU训练（速度较慢但可以运行）。

### Q: ONNX模型验证失败？
A: 检查是否安装了 `onnxruntime`：`pip install onnxruntime`

### Q: 如何查看训练历史？
A: 查看 `models/logs/training_history.json` 文件，包含每轮的详细指标。

### Q: 模型性能不够好怎么办？
A: 可以尝试：
- 调整模型结构（增加隐藏层或神经元数量）
- 调整超参数（学习率、批次大小等）
- 使用更多训练数据（增加 `SAMPLE_RATE`）

