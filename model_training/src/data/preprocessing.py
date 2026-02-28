# -*- coding: utf-8 -*-
"""
数据预处理管道：标准化、数据集划分
（按照参考文档，使用全部特征）
"""
import pandas as pd
import numpy as np
import json
import os
import sys
from sklearn.preprocessing import StandardScaler, LabelEncoder
from sklearn.model_selection import train_test_split

# 配置
DATA_DIR = os.path.join(os.path.dirname(__file__), '../../dataset')
DEV_FILE_PATH = os.path.join(DATA_DIR, 'cicids2017_dev.csv')
LABEL_MAP_PATH = os.path.join(DATA_DIR, 'label_map.json')
SCALER_PARAMS_PATH = os.path.join(DATA_DIR, 'scaler_params.json')

# 输出路径
TRAIN_PATH = os.path.join(DATA_DIR, 'train.csv')
VAL_PATH = os.path.join(DATA_DIR, 'val.csv')
TEST_PATH = os.path.join(DATA_DIR, 'test.csv')

def load_data():
    """加载数据（使用全部特征，参考文档做法）"""
    print("=" * 60)
    print("数据预处理管道")
    print("=" * 60)
    
    print("\n步骤1: 加载数据...")
    sys.stdout.flush()
    
    # 加载完整数据集
    df = pd.read_csv(DEV_FILE_PATH, low_memory=False)
    print(f"原始数据集形状: {df.shape}")
    
    # 分离特征和标签（使用全部特征，除了Label列）
    X = df.drop(columns=['Label']).copy()
    y = df['Label'].copy()
    
    print(f"特征数量: {X.shape[1]}")
    print(f"样本数量: {X.shape[0]}")
    print(f"标签分布:\n{y.value_counts()}")
    sys.stdout.flush()
    
    return X, y

def clean_data(X):
    """清洗数据：处理无效值（参考文档做法）"""
    print("\n步骤2: 数据清洗...")
    sys.stdout.flush()
    
    # 替换无穷大为NaN（参考文档第118行）
    X = X.replace([np.inf, -np.inf], np.nan)
    
    # 统计缺失值
    missing_counts = X.isnull().sum()
    if missing_counts.sum() > 0:
        print(f"发现缺失值，列数: {(missing_counts > 0).sum()}")
        print(f"缺失值总数: {missing_counts.sum()}")
    
    # 用0填充NaN（参考文档第120行）
    X = X.fillna(0)
    
    print("数据清洗完成")
    sys.stdout.flush()
    
    return X

def encode_labels(y):
    """标签编码"""
    print("\n步骤3: 标签编码...")
    sys.stdout.flush()
    
    le = LabelEncoder()
    y_encoded = le.fit_transform(y)
    
    print(f"类别数量: {len(le.classes_)}")
    print(f"类别映射: {dict(zip(le.classes_, range(len(le.classes_))))}")
    sys.stdout.flush()
    
    return y_encoded, le

def normalize_features(X_train, X_val, X_test):
    """特征标准化（参考文档第134-144行）"""
    print("\n步骤4: 特征标准化...")
    sys.stdout.flush()
    
    scaler = StandardScaler()
    
    # 只在训练集上拟合
    X_train_scaled = scaler.fit_transform(X_train)
    X_val_scaled = scaler.transform(X_val)
    X_test_scaled = scaler.transform(X_test)
    
    # 保存scaler参数（关键！Go程序需要这些参数）
    scaler_params = {
        'mean': scaler.mean_.tolist(),
        'scale': scaler.scale_.tolist(),
        'feature_names': X_train.columns.tolist()
    }
    
    with open(SCALER_PARAMS_PATH, 'w', encoding='utf-8') as f:
        json.dump(scaler_params, f, indent=4, ensure_ascii=False)
    
    print(f"标准化参数已保存至: {SCALER_PARAMS_PATH}")
    print(f"均值范围: [{scaler.mean_.min():.2f}, {scaler.mean_.max():.2f}]")
    print(f"标准差范围: [{scaler.scale_.min():.2f}, {scaler.scale_.max():.2f}]")
    sys.stdout.flush()
    
    return X_train_scaled, X_val_scaled, X_test_scaled, scaler

def split_dataset(X, y):
    """划分数据集（参考文档第126行：75%训练，25%测试）"""
    print("\n步骤5: 划分数据集...")
    sys.stdout.flush()
    
    # 按照参考文档：75%训练，25%测试
    # 但为了模型训练需要，我们进一步将测试集分为验证集和测试集
    X_train, X_temp, y_train, y_temp = train_test_split(
        X, y, test_size=0.25, random_state=42, stratify=y
    )
    
    # 将25%的测试集再分为验证集和测试集（各占12.5%）
    X_val, X_test, y_val, y_test = train_test_split(
        X_temp, y_temp, test_size=0.5, random_state=42, stratify=y_temp
    )
    
    print(f"训练集: {X_train.shape[0]} 样本 ({X_train.shape[0]/len(X)*100:.1f}%)")
    print(f"验证集: {X_val.shape[0]} 样本 ({X_val.shape[0]/len(X)*100:.1f}%)")
    print(f"测试集: {X_test.shape[0]} 样本 ({X_test.shape[0]/len(X)*100:.1f}%)")
    
    print("\n训练集标签分布:")
    print(pd.Series(y_train).value_counts().sort_index())
    sys.stdout.flush()
    
    return X_train, X_val, X_test, y_train, y_val, y_test

def save_datasets(X_train, X_val, X_test, y_train, y_val, y_test, feature_names):
    """保存处理后的数据集"""
    print("\n步骤6: 保存数据集...")
    sys.stdout.flush()
    
    # 转换为DataFrame以便保存
    train_df = pd.DataFrame(X_train, columns=feature_names)
    train_df['Label'] = y_train
    
    val_df = pd.DataFrame(X_val, columns=feature_names)
    val_df['Label'] = y_val
    
    test_df = pd.DataFrame(X_test, columns=feature_names)
    test_df['Label'] = y_test
    
    # 保存为CSV
    train_df.to_csv(TRAIN_PATH, index=False)
    val_df.to_csv(VAL_PATH, index=False)
    test_df.to_csv(TEST_PATH, index=False)
    
    print(f"训练集已保存至: {TRAIN_PATH}")
    print(f"验证集已保存至: {VAL_PATH}")
    print(f"测试集已保存至: {TEST_PATH}")
    sys.stdout.flush()

def main():
    """主函数"""
    try:
        # 1. 加载数据（使用全部特征）
        X, y = load_data()
        
        # 2. 清洗数据
        X_clean = clean_data(X)
        
        # 3. 标签编码
        y_encoded, label_encoder = encode_labels(y)
        
        # 4. 划分数据集（在标准化之前，避免数据泄露）
        X_train, X_val, X_test, y_train, y_val, y_test = split_dataset(
            X_clean, y_encoded
        )
        
        # 保存特征名（在标准化之前，因为标准化后X_train会变成numpy array）
        feature_names = X_train.columns.tolist()
        
        # 5. 特征标准化
        X_train_scaled, X_val_scaled, X_test_scaled, scaler = normalize_features(
            X_train, X_val, X_test
        )
        
        # 6. 保存数据集
        save_datasets(
            X_train_scaled, X_val_scaled, X_test_scaled,
            y_train, y_val, y_test,
            feature_names  # 使用全部特征名
        )
        
        print("\n" + "=" * 60)
        print("预处理完成！")
        print("=" * 60)
        print(f"\n产出文件:")
        print(f"  1. {TRAIN_PATH} - 训练集")
        print(f"  2. {VAL_PATH} - 验证集")
        print(f"  3. {TEST_PATH} - 测试集")
        print(f"  4. {SCALER_PARAMS_PATH} - 标准化参数（Go程序需要）")
        
    except Exception as e:
        print(f"\n错误: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()