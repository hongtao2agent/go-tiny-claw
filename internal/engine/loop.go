package engine

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/yourname/go-tiny-claw/internal/provider"
	"github.com/yourname/go-tiny-claw/internal/schema"
	"github.com/yourname/go-tiny-claw/internal/tools"
)

type AgentEngine struct {
	provider       provider.LLMProvider
	registry       tools.Registry
	WorkDir        string
	EnableThinking bool
}

func NewAgentEngine(p provider.LLMProvider, r tools.Registry, workDir string, enableThinking bool) *AgentEngine {
	return &AgentEngine{
		provider:       p,
		registry:       r,
		WorkDir:        workDir,
		EnableThinking: enableThinking,
	}
}

func (e *AgentEngine) Run(ctx context.Context, userPrompt string) error {
	return e.RunWithReporter(ctx, userPrompt, NewTerminalReporter())
}

func (e *AgentEngine) RunWithReporter(ctx context.Context, userPrompt string, reporter Reporter) error {
	log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)
	log.Printf("[Engine] 慢思考模式 (Thinking Phase): %v\n", e.EnableThinking)

	contextHistory := []schema.Message{
		{
			Role:    schema.RoleSystem,
			Content: "You are go-tiny-claw, an expert coding assistant. You have full access to tools in the workspace.",
		},
		{
			Role:    schema.RoleUser,
			Content: userPrompt,
		},
	}

	turnCount := 0
	for {
		turnCount++
		log.Printf("\n========== [Turn %d] 开始 ==========\n", turnCount)

		availableTools := e.registry.GetAvailableTools()

		if e.EnableThinking {
			log.Println("[Engine][Phase 1] 剥夺工具访问权，强制进入慢思考与规划阶段...")
			if reporter != nil {
				reporter.OnThinking(ctx)
			}

			thinkResp, err := e.provider.Generate(ctx, contextHistory, nil)
			if err != nil {
				return fmt.Errorf("Thinking 阶段生成失败: %w", err)
			}

			if thinkResp.Content != "" {
				contextHistory = append(contextHistory, *thinkResp)
			}
		}

		log.Println("[Engine][Phase 2] 恢复工具挂载，等待模型采取行动...")
		actionResp, err := e.provider.Generate(ctx, contextHistory, availableTools)
		if err != nil {
			return fmt.Errorf("Action 阶段生成失败: %w", err)
		}

		contextHistory = append(contextHistory, *actionResp)

		if actionResp.Content != "" && reporter != nil {
			reporter.OnMessage(ctx, actionResp.Content)
		}

		if len(actionResp.ToolCalls) == 0 {
			log.Println("[Engine] 模型未请求调用工具，任务宣告完成。")
			break
		}

		observationMsgs := e.executeToolCalls(ctx, actionResp.ToolCalls, reporter)
		contextHistory = append(contextHistory, observationMsgs...)
	}

	return nil
}

func (e *AgentEngine) executeToolCalls(ctx context.Context, toolCalls []schema.ToolCall, reporter Reporter) []schema.Message {
	log.Printf("[Engine] 模型请求并发调用 %d 个工具...\n", len(toolCalls))

	observationMsgs := make([]schema.Message, len(toolCalls))

	var wg sync.WaitGroup
	for i, toolCall := range toolCalls {
		wg.Add(1)
		go func(idx int, call schema.ToolCall) {
			defer wg.Done()

			log.Printf(" -> [Go-%d] 🛠️ 触发并行执行: %s, 参数: %s\n", idx, call.Name, string(call.Arguments))
			if reporter != nil {
				reporter.OnToolCall(ctx, call.Name, string(call.Arguments))
			}

			result := e.registry.Execute(ctx, call)
			if result.IsError {
				log.Printf(" -> [Go-%d] ❌ 工具执行报错: %s\n", idx, result.Output)
			} else {
				log.Printf(" -> [Go-%d] ✅ 工具执行成功 (返回 %d 字节)\n", idx, len(result.Output))
			}
			if reporter != nil {
				reporter.OnToolResult(ctx, call.Name, truncateForReporter(result.Output), result.IsError)
			}

			observationMsgs[idx] = schema.Message{
				Role:       schema.RoleUser,
				Content:    result.Output,
				ToolCallID: call.ID,
			}
		}(i, toolCall)
	}

	wg.Wait()
	log.Println("[Engine] 所有并发工具执行完毕，开始聚合观察结果 (Observation)...")

	return observationMsgs
}

func truncateForReporter(output string) string {
	const maxLen = 200
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "... (已截断)"
}
