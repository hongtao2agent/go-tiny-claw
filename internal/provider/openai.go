package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
	"github.com/yourname/go-tiny-claw/internal/schema"
)

type OpenAIProvider struct {
	client openai.Client
	model  string
}

func NewZhipuOpenAIProvider(model string) *OpenAIProvider {
	apiKey := os.Getenv("ZHIPU_API_KEY")
	if apiKey == "" {
		panic("请设置 ZHIPU_API_KEY 环境变量")
	}

	baseURL := "https://open.bigmodel.cn/api/paas/v4/"
	return &OpenAIProvider{
		client: openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL)),
		model:  model,
	}
}

func (p *OpenAIProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	var openaiMsgs []openai.ChatCompletionMessageParamUnion
	for _, msg := range msgs {
		switch msg.Role {
		case schema.RoleSystem:
			openaiMsgs = append(openaiMsgs, openai.SystemMessage(msg.Content))
		case schema.RoleUser:
			if msg.ToolCallID != "" {
				openaiMsgs = append(openaiMsgs, openai.ToolMessage(msg.Content, msg.ToolCallID))
			} else {
				openaiMsgs = append(openaiMsgs, openai.UserMessage(msg.Content))
			}
		case schema.RoleAssistant:
			astParam := openai.ChatCompletionAssistantMessageParam{}
			if msg.Content != "" {
				astParam.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(msg.Content),
				}
			}

			if len(msg.ToolCalls) > 0 {
				var toolCalls []openai.ChatCompletionMessageToolCallUnionParam
				for _, tc := range msg.ToolCalls {
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnionParam{
						OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
							ID: tc.ID,
							Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
								Name:      tc.Name,
								Arguments: string(tc.Arguments),
							},
						},
					})
				}
				astParam.ToolCalls = toolCalls
			}

			openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &astParam,
			})
		}
	}

	var openaiTools []openai.ChatCompletionToolUnionParam
	for _, toolDef := range availableTools {
		openaiTools = append(openaiTools, openai.ChatCompletionFunctionTool(
			shared.FunctionDefinitionParam{
				Name:        toolDef.Name,
				Description: openai.String(toolDef.Description),
				Parameters:  toOpenAIFunctionParameters(toolDef.InputSchema),
			},
		))
	}

	params := openai.ChatCompletionNewParams{
		Model:    p.model,
		Messages: openaiMsgs,
	}
	if len(openaiTools) > 0 {
		params.Tools = openaiTools
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI/Zhipu API 请求失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("API 返回了空的 Choices")
	}

	choice := resp.Choices[0].Message
	resultMsg := &schema.Message{
		Role:    schema.RoleAssistant,
		Content: choice.Content,
	}

	for _, tc := range choice.ToolCalls {
		if tc.Type != "function" {
			continue
		}

		resultMsg.ToolCalls = append(resultMsg.ToolCalls, schema.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: []byte(tc.Function.Arguments),
		})
	}

	return resultMsg, nil
}

func toOpenAIFunctionParameters(inputSchema interface{}) shared.FunctionParameters {
	if params, ok := inputSchema.(shared.FunctionParameters); ok {
		return params
	}

	if m, ok := inputSchema.(map[string]interface{}); ok {
		return shared.FunctionParameters(m)
	}

	b, err := json.Marshal(inputSchema)
	if err != nil {
		return nil
	}

	var params shared.FunctionParameters
	if err := json.Unmarshal(b, &params); err != nil {
		return nil
	}

	return params
}
