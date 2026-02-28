# Go IDS

Go IDS 是入侵检测系统的核心处理引擎和附带的 Web 控制台管理页面。它负责网络数据包的实时捕获、流状态重组与特征提取、利用预先加载的深度学习模型进行在线推断鉴定，并在内置的 Web 仪表板上提供可视化的威胁审查。

## 🌟 主要特性

- **并发实时数据流处理**：使用 Go 的协程特性处理 `libpcap` 捕获的流量，对数据包解码 (`decoder`) 和完整传输层连接状态 (`flow`) 进行高吞吐管理。
- **机器学习在线推理**：内部通过 CGO 调用 ONNX Runtime 引擎 (`inference`)，加载 `.onnx` 格式的深度模型，对每个提取到的网络流计算异常置信度。
- **多功能安全响应模块**：根据配置，引擎自动决策报警，并结合内建存储和策略接口进行事件日志持久化和 IP 的阻断拦截 (`response`)。
- **Web 实时态势监控板**：自带现代化 Vue 前端系统 (`web`)，经由 Go 后端提供的 REST 及服务端推送（SSE）等接口，直接展示控制台数据、威胁审计记录和配置系统。

## 📂 模块结构

```text
go_ids/
├── main.go                    # 程序入口 (位于 cmd/ids/main.go)
├── config/                    # 配置与模型文件目录
│   ├── config.yaml            # 核心引擎运行配置
│   ├── ids_model.onnx         # 预先训练好的模型权重
│   └── scaler_params.json     # 网络特征的数据标准化参数
├── internal/                  # 系统底层核心逻辑模块
│   ├── capture/               # 基于指定网卡的数据包捕获
│   ├── decoder/               # 将原始数据帧解析为协议结构
│   ├── flow/                  # 基于五元组的高效流量回话拼接与清理
│   ├── feature/               # 从重组流中抽取机器学习评估张量
│   ├── inference/             # C 接口桥接 ONNX 核心引擎
│   ├── db/                    # 高效本地 SQLite 存储支持
│   ├── response/              # 安全事件报警、阻断等联动反应
│   └── server/                # 提供给控制台界面的 REST & WebSocket 接口
├── web/                       # 前端 Vue & Vite 的控制台管理面板源码
└── onnxruntime.dll            # ONNX C++ 执行环境底层依赖
```


## 🚀 启动与构建

### 后端引擎 (Backend Engine)

1. **准备环境**：后端依赖于 Go，且在 Windows 上运行时必须带有同级目录的 `onnxruntime.dll`，在 Linux/Unix 环境需配置好对应的动态链接库。并确保系统上安装有 `Npcap`/`WinPcap` 或 `libpcap` 等抓包驱动。
2. **配置检查**：打开 `config/config.yaml` 并确认 `capture.interface` 的网卡设备句柄名是正确的。并确认模型文件及相关特征的配置文件存放于配置所在的对应路径（如 `config/ids_model.onnx`）。
3. **获取并在本地运行**：
   ```bash
   go mod download
   go run ./cmd/ids/main.go
   # 或者直接运行编译输出的 ./ids.exe 
   ```

### 前端应用 (Web Dashboard)

开发环境：
```bash
cd web
npm install
npm run dev
```

如需进行生产打包供后端静态路由服务：
```bash
cd web
npm run build
```
（构建的 `dist` 产物通常用于提供给 `server` 路由系统以便用户在正式环境中通过固定端口访问面板。）

## ⚙️ 核心配置说明 (`config/config.yaml`)

对于引擎运行的一些重要配置提示：

- **捕获与网卡侦听** (:capture`)：网卡信息可通过专门的管理工具或者设备管理器提供的设备 Guid 指定。
- **模型和预测阈值** (`detection`)：可以指定 ONNX 模型以及 JSON 结构特征缩放地图（通常被赋予如 `scaler_params.json`）的路径；`threshold` 用于断定一条通讯流是否为恶意的最终分数线。
- **响应动作策略** (`response`)：用于开启自动惩罚（封禁网络 IP 地址）、封禁时间的指定以及加入安全排除网段白名单。
)