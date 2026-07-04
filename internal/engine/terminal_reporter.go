package engine

import (
	"context"
	"fmt"
	"log"
)

type TerminalReporter struct{}

func NewTerminalReporter() *TerminalReporter {
	return &TerminalReporter{}
}

func (r *TerminalReporter) OnThinking(ctx context.Context) {
	log.Println("[Reporter] 模型正在慢思考 (Thinking)...")
}

func (r *TerminalReporter) OnToolCall(ctx context.Context, toolName string, args string) {
	log.Printf(" -> 🛠️ 执行工具: %s, 参数: %s\n", toolName, args)
}

func (r *TerminalReporter) OnToolResult(ctx context.Context, toolName string, result string, isError bool) {
	if isError {
		log.Printf(" -> ❌ 工具执行报错: %s\n", result)
		return
	}

	log.Printf(" -> ✅ 工具执行成功 (返回 %d 字节)\n", len(result))
}

func (r *TerminalReporter) OnMessage(ctx context.Context, content string) {
	fmt.Printf("🤖 [对外回复]: %s\n", content)
}
