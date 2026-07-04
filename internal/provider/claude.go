package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/yourname/go-tiny-claw/internal/schema"
)

type ClaudeProvider struct {
	client anthropic.Client
	model  string
}

func NewAnthropicCompatibleProvider(apiKey string, baseURL string, model string) *ClaudeProvider {
	if apiKey == "" {
		panic("Anthropic-compatible provider requires an API key")
	}
	if baseURL == "" {
		panic("Anthropic-compatible provider requires a base URL")
	}
	if model == "" {
		panic("Anthropic-compatible provider requires a model")
	}

	return &ClaudeProvider{
		client: anthropic.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL)),
		model:  model,
	}
}

func NewMiniMaxAnthropicProvider(model string) *ClaudeProvider {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		panic("请设置 MINIMAX_API_KEY 或 ANTHROPIC_API_KEY 环境变量")
	}

	baseURL := os.Getenv("MINIMAX_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com/anthropic"
	}

	if model == "" {
		model = "MiniMax-M3"
	}

	return NewAnthropicCompatibleProvider(apiKey, baseURL, model)
}

func NewZhipuClaudeProvider(model string) *ClaudeProvider {
	apiKey := os.Getenv("ZHIPU_API_KEY")
	if apiKey == "" {
		panic("请设置 ZHIPU_API_KEY 环境变量")
	}

	baseURL := "https://open.bigmodel.cn/api/paas/v4/"
	return NewAnthropicCompatibleProvider(apiKey, baseURL, model)
}

func (p *ClaudeProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	var anthropicMsgs []anthropic.MessageParam
	var systemPrompt string

	for _, msg := range msgs {
		switch msg.Role {
		case schema.RoleSystem:
			systemPrompt = msg.Content
		case schema.RoleUser:
			if msg.ToolCallID != "" {
				anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
				))
			} else {
				anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
					anthropic.NewTextBlock(msg.Content),
				))
			}
		case schema.RoleAssistant:
			var blocks []anthropic.ContentBlockParamUnion
			if msg.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
			}

			for _, tc := range msg.ToolCalls {
				var inputMap map[string]interface{}
				_ = json.Unmarshal(tc.Arguments, &inputMap)

				blocks = append(blocks, anthropic.ContentBlockParamUnion{
					OfToolUse: &anthropic.ToolUseBlockParam{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: inputMap,
					},
				})
			}

			if len(blocks) > 0 {
				anthropicMsgs = append(anthropicMsgs, anthropic.NewAssistantMessage(blocks...))
			}
		}
	}

	var anthropicTools []anthropic.ToolUnionParam
	for _, toolDef := range availableTools {
		toolParam := anthropic.ToolParam{
			Name:        toolDef.Name,
			Description: anthropic.String(toolDef.Description),
			InputSchema: toAnthropicToolInputSchema(toolDef.InputSchema),
		}

		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{OfTool: &toolParam})
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 4096,
		Messages:  anthropicMsgs,
	}
	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{{Text: systemPrompt}}
	}
	if len(anthropicTools) > 0 {
		params.Tools = anthropicTools
	}

	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("Anthropic-compatible API 请求失败: %w", err)
	}

	resultMsg := &schema.Message{
		Role: schema.RoleAssistant,
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			resultMsg.Content += block.Text
		case "tool_use":
			resultMsg.ToolCalls = append(resultMsg.ToolCalls, schema.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}

	return resultMsg, nil
}

func toAnthropicToolInputSchema(inputSchema interface{}) anthropic.ToolInputSchemaParam {
	m := schemaMap(inputSchema)

	result := anthropic.ToolInputSchemaParam{}
	if properties, ok := m["properties"]; ok {
		result.Properties = properties
	}
	result.Required = stringSlice(m["required"])

	return result
}

func schemaMap(inputSchema interface{}) map[string]interface{} {
	if m, ok := inputSchema.(map[string]interface{}); ok {
		return m
	}

	b, err := json.Marshal(inputSchema)
	if err != nil {
		return nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}

	return m
}

func stringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		items := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
		return items
	default:
		return nil
	}
}
