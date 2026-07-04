# go-tiny-claw

`go-tiny-claw` 是一个从 0 开始构建 Agent Harness 的 Go 语言练习项目。

当前骨架按照文章中的四层架构初始化：

- `cmd/claw`: CLI 入口
- `internal/engine`: 核心 ReAct 循环与运行时
- `internal/provider`: 大模型适配器
- `internal/schema`: 消息、工具调用与工具结果协议
- `internal/context`: 上下文工程
- `internal/tools`: 工具注册、分发与执行
- `internal/memory`: 文件系统状态与记忆
- `internal/feishu`: 飞书集成

运行入口：

```bash
go run cmd/claw/main.go
```

两阶段 ReAct 的核心流程：

- Thinking 阶段：不挂载工具，强制模型输出内部规划。
- Action 阶段：恢复工具挂载，根据规划决定是否执行工具。

默认入口已接入 MiniMax Anthropic 兼容 Provider，默认模型为 `MiniMax-M3`，并挂载真实 `read_file`、`write_file`、`bash`、`edit_file` 工具。运行前需要设置：

```bash
export MINIMAX_API_KEY=你的 MiniMax API Key
go run cmd/claw/main.go
```

也兼容官方 Anthropic SDK 环境变量名：

```bash
export ANTHROPIC_API_KEY=你的 MiniMax API Key
export MINIMAX_BASE_URL=https://api.minimaxi.com/anthropic
go run cmd/claw/main.go
```

可选切换模型与慢思考模式：

```bash
export MINIMAX_MODEL=MiniMax-M3
export TINY_CLAW_ENABLE_THINKING=true
go run cmd/claw/main.go
```

默认 Prompt 会要求模型在同一轮里发起三个 `read_file` 工具调用读取 `README.md`、`go.mod`、`deploy/README.md`，验证单轮多工具并发调用。默认开启两阶段慢思考；如需关闭：

```bash
export TINY_CLAW_ENABLE_THINKING=false
go run cmd/claw/main.go
```

也可以覆盖 Prompt：

```bash
export TINY_CLAW_PROMPT="请读取 README.md 并总结当前项目结构"
go run cmd/claw/main.go
```

如需回到第 04 讲中的智谱示例：

```bash
export TINY_CLAW_PROVIDER=zhipu-openai
export ZHIPU_API_KEY=你的智谱 API Key
go run cmd/claw/main.go
```

## 飞书模式

第 09 讲后，入口支持飞书事件回调模式。大模型仍默认使用 MiniMax Anthropic 兼容接口和 `MiniMax-M3`。

需要在飞书开放平台创建企业自建应用，启用机器人能力，并配置事件订阅。服务端启动后，把公网可访问的回调地址填到飞书事件订阅中：

```text
http://你的公网域名或IP:48080/webhook/event
```

启动服务：

```bash
export TINY_CLAW_MODE=feishu
export MINIMAX_API_KEY=你的 MiniMax API Key

export FEISHU_APP_ID=cli_xxx
export FEISHU_APP_SECRET=xxx
export FEISHU_VERIFY_TOKEN=xxx
export FEISHU_ENCRYPT_KEY=xxx

go run cmd/claw/main.go
```

可选配置：

```bash
export TINY_CLAW_HTTP_ADDR=:48080
export TINY_CLAW_ENABLE_THINKING=true
```

收到飞书文本消息后，机器人会启动一轮 Agent Run，并把思考状态、工具调用、工具结果和最终回复发送回同一个会话。
