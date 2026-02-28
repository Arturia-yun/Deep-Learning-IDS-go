# -*- coding: utf-8 -*-
"""
神经网络模型定义
"""
import torch
import torch.nn as nn
import torch.nn.functional as F

class IDSClassifier(nn.Module):
    """
    入侵检测分类器
    使用多层感知机（MLP）结构
    """
    def __init__(self, input_dim=78, hidden_dims=[128, 64, 32], num_classes=6, dropout_rate=0.3):
        """
        初始化模型
        
        Args:
            input_dim: 输入特征维度（78个特征）
            hidden_dims: 隐藏层维度列表
            num_classes: 输出类别数（6类）
            dropout_rate: Dropout比率
        """
        super(IDSClassifier, self).__init__()
        
        # 构建全连接层
        layers = []
        prev_dim = input_dim
        
        for hidden_dim in hidden_dims:
            layers.append(nn.Linear(prev_dim, hidden_dim))
            layers.append(nn.ReLU())
            layers.append(nn.Dropout(dropout_rate))
            prev_dim = hidden_dim
        
        # 输出层
        layers.append(nn.Linear(prev_dim, num_classes))
        
        self.network = nn.Sequential(*layers)
    
    def forward(self, x):
        """
        前向传播
        
        Args:
            x: 输入特征张量 [batch_size, input_dim]
        
        Returns:
            输出logits [batch_size, num_classes]
        """
        return self.network(x)
    
    def predict_proba(self, x):
        """
        预测概率
        
        Args:
            x: 输入特征张量
        
        Returns:
            类别概率 [batch_size, num_classes]
        """
        with torch.no_grad():
            logits = self.forward(x)
            return F.softmax(logits, dim=1)
    
    def predict(self, x):
        """
        预测类别
        
        Args:
            x: 输入特征张量
        
        Returns:
            预测类别 [batch_size]
        """
        proba = self.predict_proba(x)
        return torch.argmax(proba, dim=1)

def create_model(input_dim=78, num_classes=6, **kwargs):
    """
    创建模型实例的便捷函数
    
    Args:
        input_dim: 输入特征维度
        num_classes: 输出类别数
        **kwargs: 其他模型参数
    
    Returns:
        模型实例
    """
    return IDSClassifier(input_dim=input_dim, num_classes=num_classes, **kwargs)

