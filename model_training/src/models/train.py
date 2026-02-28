# -*- coding: utf-8 -*-
"""
模型训练脚本
"""
import torch
import torch.nn as nn
import torch.optim as optim
from torch.utils.data import Dataset, DataLoader
import pandas as pd
import numpy as np
import os
import sys
import json
from sklearn.metrics import accuracy_score, precision_recall_fscore_support, classification_report, confusion_matrix

# 添加父目录到路径，以便导入模型
sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))
from models.model import create_model

# 配置
DATA_DIR = os.path.join(os.path.dirname(__file__), '../../dataset')
TRAIN_PATH = os.path.join(DATA_DIR, 'train.csv')
VAL_PATH = os.path.join(DATA_DIR, 'val.csv')
MODEL_DIR = os.path.join(os.path.dirname(__file__), '../../models')
CHECKPOINT_DIR = os.path.join(MODEL_DIR, 'checkpoints')
LOG_DIR = os.path.join(MODEL_DIR, 'logs')
BEST_MODEL_PATH = os.path.join(CHECKPOINT_DIR, 'best_model.pth')
LABEL_MAP_PATH = os.path.join(DATA_DIR, 'label_map.json')

# 训练超参数
BATCH_SIZE = 256
LEARNING_RATE = 0.001
NUM_EPOCHS = 50
DEVICE = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
EARLY_STOPPING_PATIENCE = 10  # 早停耐心值

class IDSDataset(Dataset):
    """入侵检测数据集"""
    def __init__(self, csv_path):
        """
        初始化数据集
        
        Args:
            csv_path: CSV文件路径
        """
        df = pd.read_csv(csv_path)
        
        # 分离特征和标签
        self.features = df.drop(columns=['Label']).values.astype(np.float32)
        self.labels = df['Label'].values.astype(np.int64)
        
        print(f"加载数据集: {csv_path}")
        print(f"  样本数: {len(self.features)}")
        print(f"  特征数: {self.features.shape[1]}")
        print(f"  类别数: {len(np.unique(self.labels))}")
    
    def __len__(self):
        return len(self.features)
    
    def __getitem__(self, idx):
        return torch.tensor(self.features[idx]), torch.tensor(self.labels[idx])

def train_epoch(model, dataloader, criterion, optimizer, device):
    """训练一个epoch"""
    model.train()
    total_loss = 0
    all_preds = []
    all_labels = []
    
    for features, labels in dataloader:
        features = features.to(device)
        labels = labels.to(device)
        
        # 前向传播
        optimizer.zero_grad()
        outputs = model(features)
        loss = criterion(outputs, labels)
        
        # 反向传播
        loss.backward()
        optimizer.step()
        
        # 统计
        total_loss += loss.item()
        preds = torch.argmax(outputs, dim=1)
        all_preds.extend(preds.cpu().numpy())
        all_labels.extend(labels.cpu().numpy())
    
    avg_loss = total_loss / len(dataloader)
    accuracy = accuracy_score(all_labels, all_preds)
    
    return avg_loss, accuracy

def validate(model, dataloader, criterion, device):
    """验证模型"""
    model.eval()
    total_loss = 0
    all_preds = []
    all_labels = []
    
    with torch.no_grad():
        for features, labels in dataloader:
            features = features.to(device)
            labels = labels.to(device)
            
            outputs = model(features)
            loss = criterion(outputs, labels)
            
            total_loss += loss.item()
            preds = torch.argmax(outputs, dim=1)
            all_preds.extend(preds.cpu().numpy())
            all_labels.extend(labels.cpu().numpy())
    
    avg_loss = total_loss / len(dataloader)
    accuracy = accuracy_score(all_labels, all_preds)
    precision, recall, f1, _ = precision_recall_fscore_support(
        all_labels, all_preds, average='weighted', zero_division=0
    )
    
    return avg_loss, accuracy, precision, recall, f1

def main():
    """主训练函数"""
    print("=" * 60)
    print("模型训练")
    print("=" * 60)
    print(f"使用设备: {DEVICE}")
    print(f"批次大小: {BATCH_SIZE}")
    print(f"学习率: {LEARNING_RATE}")
    print(f"训练轮数: {NUM_EPOCHS}")
    print()
    
    # 创建模型输出目录
    os.makedirs(CHECKPOINT_DIR, exist_ok=True)
    os.makedirs(LOG_DIR, exist_ok=True)
    
    # 加载数据集
    print("加载数据集...")
    train_dataset = IDSDataset(TRAIN_PATH)
    val_dataset = IDSDataset(VAL_PATH)
    
    train_loader = DataLoader(train_dataset, batch_size=BATCH_SIZE, shuffle=True, num_workers=0)
    val_loader = DataLoader(val_dataset, batch_size=BATCH_SIZE, shuffle=False, num_workers=0)
    
    # 创建模型
    input_dim = train_dataset.features.shape[1]
    num_classes = len(np.unique(train_dataset.labels))
    
    print(f"\n创建模型...")
    print(f"  输入维度: {input_dim}")
    print(f"  类别数: {num_classes}")
    
    model = create_model(input_dim=input_dim, num_classes=num_classes)
    model = model.to(DEVICE)
    
    # 损失函数和优化器
    criterion = nn.CrossEntropyLoss()
    optimizer = optim.Adam(model.parameters(), lr=LEARNING_RATE)
    scheduler = optim.lr_scheduler.ReduceLROnPlateau(optimizer, mode='min', factor=0.5, patience=5)
    
    # 训练循环
    best_val_loss = float('inf')
    best_val_f1 = 0.0
    patience_counter = 0
    train_history = []
    
    print("\n开始训练...")
    print("-" * 60)
    
    for epoch in range(NUM_EPOCHS):
        # 训练
        train_loss, train_acc = train_epoch(model, train_loader, criterion, optimizer, DEVICE)
        
        # 验证
        val_loss, val_acc, val_precision, val_recall, val_f1 = validate(model, val_loader, criterion, DEVICE)
        
        # 学习率调度
        scheduler.step(val_loss)
        
        # 记录历史
        train_history.append({
            'epoch': epoch + 1,
            'train_loss': train_loss,
            'train_acc': train_acc,
            'val_loss': val_loss,
            'val_acc': val_acc,
            'val_precision': val_precision,
            'val_recall': val_recall,
            'val_f1': val_f1
        })
        
        # 打印进度
        print(f"Epoch [{epoch+1}/{NUM_EPOCHS}]")
        print(f"  Train Loss: {train_loss:.4f}, Train Acc: {train_acc:.4f}")
        print(f"  Val Loss: {val_loss:.4f}, Val Acc: {val_acc:.4f}, Val F1: {val_f1:.4f}")
        
        # 保存最佳模型
        if val_f1 > best_val_f1:
            best_val_f1 = val_f1
            best_val_loss = val_loss
            torch.save({
                'epoch': epoch + 1,
                'model_state_dict': model.state_dict(),
                'optimizer_state_dict': optimizer.state_dict(),
                'val_loss': val_loss,
                'val_f1': val_f1,
                'input_dim': input_dim,
                'num_classes': num_classes,
            }, BEST_MODEL_PATH)
            print(f"  ✓ 保存最佳模型 (F1: {val_f1:.4f})")
            patience_counter = 0
        else:
            patience_counter += 1
        
        # 早停
        if patience_counter >= EARLY_STOPPING_PATIENCE:
            print(f"\n早停触发 (耐心值: {EARLY_STOPPING_PATIENCE})")
            break
        
        print()
    
    # 加载最佳模型进行最终评估
    print("=" * 60)
    print("加载最佳模型进行最终评估...")
    checkpoint = torch.load(BEST_MODEL_PATH)
    model.load_state_dict(checkpoint['model_state_dict'])
    
    val_loss, val_acc, val_precision, val_recall, val_f1 = validate(model, val_loader, criterion, DEVICE)
    
    print("\n最终验证集性能:")
    print(f"  准确率 (Accuracy): {val_acc:.4f}")
    print(f"  精确率 (Precision): {val_precision:.4f}")
    print(f"  召回率 (Recall): {val_recall:.4f}")
    print(f"  F1分数 (F1-Score): {val_f1:.4f}")
    
    # 保存训练历史
    history_path = os.path.join(LOG_DIR, 'training_history.json')
    with open(history_path, 'w', encoding='utf-8') as f:
        json.dump(train_history, f, indent=4, ensure_ascii=False)
    print(f"\n训练历史已保存至: {history_path}")
    print(f"最佳模型已保存至: {BEST_MODEL_PATH}")
    
    print("\n" + "=" * 60)
    print("训练完成！")
    print("=" * 60)

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"\n错误: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)

