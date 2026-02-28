import pandas as pd
import numpy as np
import glob
import os
import json

# 配置
RAW_DATA_DIR = os.path.join(os.path.dirname(__file__), '../../dataset/MachineLearningCVE')
PROCESSED_DATA_DIR = os.path.join(os.path.dirname(__file__), '../../dataset')
DEV_FILE_PATH = os.path.join(PROCESSED_DATA_DIR, 'cicids2017_dev.csv')
LABEL_MAP_PATH = os.path.join(PROCESSED_DATA_DIR, 'label_map.json')
SAMPLE_RATE = 0.1  # 保留 10% 的数据用于开发
RANDOM_STATE = 42

def load_and_process_data():
    print("Step 1: 加载并合并 CSV 文件...")
    all_files = glob.glob(os.path.join(RAW_DATA_DIR, "*.csv"))
    if not all_files:
        print(f"错误: 在 {RAW_DATA_DIR} 中未找到 CSV 文件")
        return

    df_list = []
    for filename in all_files:
        print(f"正在读取 {os.path.basename(filename)}...")
        try:
            # 读取 csv，处理潜在的编码问题
            df_temp = pd.read_csv(filename, encoding='cp1252', low_memory=False) 
            # 基础清洗：去除列名中的空格
            df_temp.columns = df_temp.columns.str.strip()
            df_list.append(df_temp)
        except Exception as e:
            print(f"读取失败 {filename}: {e}")

    if not df_list:
        print("未加载任何数据。")
        return

    df = pd.concat(df_list, ignore_index=True)
    print(f"原始记录总数: {df.shape[0]}")

    print("Step 2: 基础清洗与标签映射...")
    # 清洗列名（再次去除空格以防万一）
    df.columns = df.columns.str.strip()
    
    # 修复标签拼写错误和编码问题
    # 基于参考文档和常见的 CIC-IDS2017 问题
    # 注意：这里只做初步的字符串替换，后续会进行更健壮的包含匹配
    df['Label'] = df['Label'].replace({
        'Web Attack  Brute Force': 'Web Attack',
        'Web Attack  XSS': 'Web Attack',
        'Web Attack  Sql Injection': 'Web Attack',
        'DoS slowloris': 'DoS',
        'DoS Slowhttptest': 'DoS',
        'DoS Hulk': 'DoS',
        'DoS GoldenEye': 'DoS',
        'Heartbleed': 'DoS',
        'FTP-Patator': 'Brute Force',
        'SSH-Patator': 'Brute Force',
        'Infiltration': 'PortScan', # 参考文档将 Infiltration 映射为 PortScan，我们遵循此指令
        'Bot': 'Bot',
        'PortScan': 'PortScan',
        'DDoS': 'DoS', # 将 DDoS 归类为 DoS 以简化处理，或保持独立？文档建议归类。
        'BENIGN': 'Benign'
    })

    # 标准化为 7 类（实际处理中可能少于 7 类）
    # 参考文档分类：Normal(Benign), DoS, PortScan, Patator(BruteForce), Web Attack, Bot, Infiltration
    
    # 处理 BENIGN
    df['Label'] = df['Label'].replace('BENIGN', 'Benign')

    # 移除标签中的不可打印字符，这些字符可能导致编码错误
    df['Label'] = df['Label'].apply(lambda x: ''.join([i for i in str(x) if i.isprintable()]))

    # 修复特定的问题标签，即使 replace 没有完全匹配，使用包含匹配更稳健
    df.loc[df['Label'].str.contains('Web Attack', case=False, na=False), 'Label'] = 'Web Attack'
    df.loc[df['Label'].str.contains('DoS', case=False, na=False), 'Label'] = 'DoS'
    df.loc[df['Label'].str.contains('Heartbleed', case=False, na=False), 'Label'] = 'DoS'
    df.loc[df['Label'].str.contains('Patator', case=False, na=False), 'Label'] = 'Brute Force'
    df.loc[df['Label'].str.contains('Infiltration', case=False, na=False), 'Label'] = 'PortScan'
    df.loc[df['Label'].str.contains('Bot', case=False, na=False), 'Label'] = 'Bot'
    df.loc[df['Label'].str.contains('PortScan', case=False, na=False), 'Label'] = 'PortScan'

    print("映射后的唯一标签 (安全编码输出):")
    for label in df['Label'].unique():
        try:
            print(label)
        except UnicodeEncodeError:
            print(label.encode('utf-8', 'ignore'))

    print("Step 3: 分层采样...")
    # 删除标签为 NaN/Inf 的行（如果有）
    df = df.dropna(subset=['Label'])

    # 分层采样
    try:
        # 对稀有类别使用最小样本量以避免错误
        # group_keys=False 避免索引层级增加
        df_dev = df.groupby('Label', group_keys=False).apply(lambda x: x.sample(frac=SAMPLE_RATE, random_state=RANDOM_STATE))
    except Exception as e:
        print(f"采样失败: {e}。回退到简单头部截取。")
        df_dev = df.head(int(len(df) * SAMPLE_RATE))

    print(f"开发集大小: {df_dev.shape[0]}")
    print("开发集标签分布:")
    print(df_dev['Label'].value_counts())

    print("Step 4: 保存开发数据集...")
    df_dev.to_csv(DEV_FILE_PATH, index=False)
    print(f"已保存至 {DEV_FILE_PATH}")

    # 保存标签映射以供将来参考 (Go 实现)
    # 创建整数映射
    unique_labels = sorted(df_dev['Label'].unique())
    label_map = {label: idx for idx, label in enumerate(unique_labels)}
    
    with open(LABEL_MAP_PATH, 'w') as f:
        json.dump(label_map, f, indent=4)
    print(f"已保存标签映射至 {LABEL_MAP_PATH}")

if __name__ == "__main__":
    load_and_process_data()

