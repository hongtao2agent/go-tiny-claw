package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/larksuite/oapi-sdk-go/v3/core/httpserverext"
	"github.com/yourname/go-tiny-claw/internal/engine"
	"github.com/yourname/go-tiny-claw/internal/feishu"
	"github.com/yourname/go-tiny-claw/internal/provider"
	"github.com/yourname/go-tiny-claw/internal/tools"
)

func main() {
	log.SetOutput(os.Stdout)

	eng := buildAgentEngine()

	switch strings.ToLower(envOrDefault("TINY_CLAW_MODE", "cli")) {
	case "cli", "terminal":
		runCLI(eng)
	case "feishu", "server":
		runFeishuServer(eng)
	default:
		log.Fatal("TINY_CLAW_MODE 只支持 cli 或 feishu")
	}
}

func buildAgentEngine() *engine.AgentEngine {
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("获取工作区失败: %v", err)
	}

	var llmProvider provider.LLMProvider
	switch strings.ToLower(os.Getenv("TINY_CLAW_PROVIDER")) {
	case "", "minimax", "minimax-anthropic", "anthropic":
		if os.Getenv("MINIMAX_API_KEY") == "" && os.Getenv("ANTHROPIC_API_KEY") == "" {
			log.Fatal("请先导出 MINIMAX_API_KEY 或 ANTHROPIC_API_KEY 环境变量")
		}
		llmProvider = provider.NewMiniMaxAnthropicProvider(envOrDefault("MINIMAX_MODEL", "MiniMax-M3"))
	case "openai", "zhipu-openai":
		if os.Getenv("ZHIPU_API_KEY") == "" {
			log.Fatal("请先导出 ZHIPU_API_KEY 环境变量")
		}
		llmProvider = provider.NewZhipuOpenAIProvider("glm-4.5-air")
	case "claude", "zhipu-claude":
		if os.Getenv("ZHIPU_API_KEY") == "" {
			log.Fatal("请先导出 ZHIPU_API_KEY 环境变量")
		}
		llmProvider = provider.NewZhipuClaudeProvider("glm-4.5-air")
	default:
		log.Fatal("TINY_CLAW_PROVIDER 只支持 minimax、openai/zhipu-openai 或 claude/zhipu-claude")
	}

	registry := tools.NewRegistry()
	registry.Register(tools.NewReadFileTool(workDir))
	registry.Register(tools.NewWriteFileTool(workDir))
	registry.Register(tools.NewBashTool(workDir))
	registry.Register(tools.NewEditFileTool(workDir))

	enableThinking := !strings.EqualFold(os.Getenv("TINY_CLAW_ENABLE_THINKING"), "false")
	return engine.NewAgentEngine(llmProvider, registry, workDir, enableThinking)
}

func runCLI(eng *engine.AgentEngine) {
	prompt := envOrDefault("TINY_CLAW_PROMPT", `我当前目录下有 README.md、go.mod、deploy/README.md 三个文件。

为了节省时间，请你在同一个回复里一次性发起三个 read_file 工具调用，分别读取 README.md、go.mod、deploy/README.md。不要使用 bash。读取后将它们的内容综合起来，告诉我这个项目的用途、依赖和部署方式。`)
	if err := eng.Run(context.Background(), prompt); err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}

func runFeishuServer(eng *engine.AgentEngine) {
	bot, err := feishu.NewFeishuBotFromEnv(eng)
	if err != nil {
		log.Fatalf("初始化飞书 Bot 失败: %v", err)
	}

	mux := http.NewServeMux()
	handler := httpserverext.NewEventHandlerFunc(bot.GetEventDispatcher())
	mux.HandleFunc("/webhook/event", handler)

	addr := envOrDefault("TINY_CLAW_HTTP_ADDR", ":48080")
	log.Printf("🚀 go-tiny-claw 飞书服务端已启动，正在监听 %s，事件路径 /webhook/event\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
