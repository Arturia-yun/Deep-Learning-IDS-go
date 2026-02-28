# Deep Learning IDS 

![Project Stage](https://img.shields.io/badge/Status-Development-yellow.svg)
![Vue](https://img.shields.io/badge/Frontend-Vue_3-4FC08D?logo=vue.js)
![Go](https://img.shields.io/badge/Engine-Go_1.19+-00ADD8?logo=go)
![PyTorch](https://img.shields.io/badge/Model-PyTorch_&_ONNX-EE4C2C?logo=pytorch)

Deep Learning IDS 是一个以深度学习（Deep Learning）为驱动的**高性能网络入侵检测系统**。系统通过离线分析海量网络数据集（如 CIC-IDS2017）训练出能够识别潜在攻击的网络模型，并利用基于 Go 语言构建的底层高性能抓包引擎捕获当前设备的网络会话状态，将实时流量转换并注入到编译后的 ONNX 智能引擎中打分预警。

不仅拥有高效的数据截获能力与机器学习底座，项目还配备了基于 Vue.js 的现代化 Web 控制面板，让非安全专业的操作人员能够直观清晰地追踪和响应网络威胁。

## 🎯 系统架构概览

本项目整体上分为离线的“模型开发栈”与在线的“运行时推理栈”，由两个主要模块组成：

```text
Deep-Learning-IDS-go/
├── model_training/            # 🧠 离线模型训练系统 (Python 环境)
│   ├── dataset/               # 特征数据集存储与拆分
│   ├── src/                   # 包含数据预处理、多层感知机（MLP）训练等流水线代码
│   └── models/                # 训练过程 Checkpoints 及最终导出的跨平台 ONNX 权重文件
│
└── go_ids/                    # 🚀 在线推理响应与前台控制界面 (Go 环境)
    ├── main.go                # 检测引擎核心服务端入口
    ├── internal/              # 底层数据包解析 (capture/decoder)，流量重组特征化 (flow/feature) 逻辑
    ├── config/                # 接收模型配置、特征标准参数的部署位置
    ├── onnxruntime.dll        # （取决于系统）底层的深度学习 C++ 链接库支持
    └── web/                   # 💻 前端面板系统 (Vue 3 + Vite) 提供预警拦截仪表板可视化
```


## 🌟 核心能力亮点

1. **从协议解析到语义转换**（`go_ids/internal/decoder & flow`）
   通过基于底层 pcap 库截获局域网及主机网卡的实时报文流，在 Go 的高性能并发体系下自动重组碎裂的分组包并还原 TCP/UDP 等协议的完整“流回话”（Flow），计算例如数据包时间间隔差异、标志位状况等维度特征。
   
2. **离线的算法探索与迭代**（`model_training/`）
   利用 PyTorch 构建经典的多层神经网络，借助 CIC-IDS2017 规模化数据集验证诸如：DDoS, Web Attack, PortScan 等 6 大类攻击模型。它封装了特征的缩放逻辑，并自动追踪评价 F1/Accuracy 训练打分情况。
   
3. **低延迟的端到端侦测**（`go_ids/internal/inference`）
   在流量捕获中直接利用 ONNX Runtime 提供的 CGO 对接接口，省去了笨重的 Python 环境。即时地对从网卡收集上的数据计算异常信心分数，以实现毫秒级别的风险阻断。

4. **现代化响应与追踪手段**（`go_ids/web`）
   配合内置 SQLite 保存的长期通讯日志与系统分析结果，控制面板提供了诸如：“最新网络流”、“威胁等级分布”和“阻断策略管理”等直观界面，为企业级和个人极客化部署提供了即时反应途径。

## 🚀 模块使用指引

这里是对两部分的快速索引摘要，详细的使用开发方式请移步至各模块下的专属 README 文档：

- ⚙️ **模型训练指引**：如果您想基于新的抓包数据集，重新进行清洗并训练深度网络得到定制的模型 `.onnx`，请参考 [模型训练与导出说明 (model_training)](model_training/README.md)。
- 🛡️ **安全引擎与界面指引**：如果您想要把现成的模型部署在服务器或者自己的计算机上，修改检测网卡的源接口或启动 Web 服务界面，请参考 [主控程序与面板部署 (go_ids)](go_ids/README.md)。

---
*关于本系统的具体性能测评与网络攻击分类拦截实现机制细节等，可访问内置系统代码源文件中附带的报告记录获取更多信息。*
