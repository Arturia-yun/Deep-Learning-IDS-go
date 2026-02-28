# -*- coding: utf-8 -*-
"""
模型导出为ONNX格式
"""
import torch
import torch.onnx
import os
import sys
import numpy as np
import onnxruntime as ort

# 添加父目录到路径
sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))
from models.model import create_model

# 配置
MODEL_DIR = os.path.join(os.path.dirname(__file__), '../../models')
CHECKPOINT_DIR = os.path.join(MODEL_DIR, 'checkpoints')
ONNX_DIR = os.path.join(MODEL_DIR, 'onnx')
BEST_MODEL_PATH = os.path.join(CHECKPOINT_DIR, 'best_model.pth')
ONNX_MODEL_PATH = os.path.join(ONNX_DIR, 'ids_model.onnx')
DATA_DIR = os.path.join(os.path.dirname(__file__), '../../dataset')
TEST_PATH = os.path.join(DATA_DIR, 'test.csv')

def load_model(checkpoint_path):
    """加载训练好的模型"""
    print("加载模型检查点...")
    checkpoint = torch.load(checkpoint_path, map_location='cpu')
    
    input_dim = checkpoint['input_dim']
    num_classes = checkpoint['num_classes']
    
    model = create_model(input_dim=input_dim, num_classes=num_classes)
    model.load_state_dict(checkpoint['model_state_dict'])
    model.eval()
    
    print(f"  输入维度: {input_dim}")
    print(f"  类别数: {num_classes}")
    print(f"  训练轮数: {checkpoint['epoch']}")
    print(f"  验证集F1: {checkpoint['val_f1']:.4f}")
    
    return model, input_dim

def export_to_onnx(model, input_dim, output_path):
    """导出模型为ONNX格式"""
    print("\n导出ONNX模型...")
    
    # 创建示例输入
    dummy_input = torch.randn(1, input_dim)
    
    # 导出
    torch.onnx.export(
        model,
        dummy_input,
        output_path,
        input_names=['features'],
        output_names=['logits'],
        dynamic_axes={
            'features': {0: 'batch_size'},
            'logits': {0: 'batch_size'}
        },
        opset_version=11,
        do_constant_folding=True,
        verbose=False
    )
    
    print(f"模型已导出至: {output_path}")

def verify_onnx_model(onnx_path, test_data_path, num_samples=100):
    """验证ONNX模型与PyTorch模型输出一致"""
    print("\n验证ONNX模型...")
    
    # 加载测试数据
    import pandas as pd
    df = pd.read_csv(test_data_path, nrows=num_samples)
    test_features = df.drop(columns=['Label']).values.astype(np.float32)
    test_labels = df['Label'].values
    
    # 加载PyTorch模型
    model, input_dim = load_model(BEST_MODEL_PATH)
    model.eval()
    
    # PyTorch预测
    with torch.no_grad():
        pytorch_input = torch.tensor(test_features)
        pytorch_output = model(pytorch_input)
        pytorch_proba = torch.softmax(pytorch_output, dim=1).numpy()
        pytorch_preds = np.argmax(pytorch_proba, axis=1)
    
    # ONNX Runtime预测
    session = ort.InferenceSession(onnx_path)
    input_name = session.get_inputs()[0].name
    
    onnx_output = session.run(None, {input_name: test_features})[0]
    onnx_proba = np.exp(onnx_output) / np.sum(np.exp(onnx_output), axis=1, keepdims=True)  # Softmax
    onnx_preds = np.argmax(onnx_proba, axis=1)
    
    # 比较结果
    pred_match = np.sum(pytorch_preds == onnx_preds)
    proba_diff = np.abs(pytorch_proba - onnx_proba).max()
    
    print(f"  测试样本数: {num_samples}")
    print(f"  预测一致率: {pred_match}/{num_samples} ({pred_match/num_samples*100:.2f}%)")
    print(f"  概率最大差异: {proba_diff:.6f}")
    
    if pred_match == num_samples and proba_diff < 1e-5:
        print("  ✓ ONNX模型验证通过！")
        return True
    else:
        print("  ✗ ONNX模型验证失败！")
        return False

def main():
    """主函数"""
    print("=" * 60)
    print("模型导出为ONNX格式")
    print("=" * 60)
    
    # 检查模型文件是否存在
    if not os.path.exists(BEST_MODEL_PATH):
        print(f"错误: 未找到模型文件 {BEST_MODEL_PATH}")
        print("请先运行 train.py 训练模型")
        sys.exit(1)
    
    # 创建输出目录
    os.makedirs(ONNX_DIR, exist_ok=True)
    
    # 加载模型
    model, input_dim = load_model(BEST_MODEL_PATH)
    
    # 导出ONNX
    export_to_onnx(model, input_dim, ONNX_MODEL_PATH)
    
    # 验证ONNX模型
    if os.path.exists(TEST_PATH):
        verify_onnx_model(ONNX_MODEL_PATH, TEST_PATH, num_samples=100)
    else:
        print("\n警告: 未找到测试集，跳过ONNX模型验证")
    
    print("\n" + "=" * 60)
    print("导出完成！")
    print("=" * 60)
    print(f"\n产出文件:")
    print(f"  1. {ONNX_MODEL_PATH} - ONNX模型文件（Go程序使用）")
    print(f"  2. {BEST_MODEL_PATH} - PyTorch模型检查点")
    print(f"\n文件位置:")
    print(f"  - PyTorch检查点: {CHECKPOINT_DIR}/")
    print(f"  - ONNX模型: {ONNX_DIR}/")

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"\n错误: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)

