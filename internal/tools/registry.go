package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

type BaseTool interface {
	Name() string
	Definition() schema.ToolDefinition
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

type Registry interface {
	Register(tool BaseTool)
	GetAvailableTools() []schema.ToolDefinition
	Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult
}

type registryImpl struct {
	tools map[string]BaseTool
}

func NewRegistry() Registry {
	return &registryImpl{
		tools: make(map[string]BaseTool),
	}
}

func (r *registryImpl) Register(tool BaseTool) {
	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		log.Printf("[Warning] 工具 '%s' 已经被注册，将被覆盖。\n", name)
	}

	r.tools[name] = tool
	log.Printf("[Registry] 成功挂载工具: %s\n", name)
}

func (r *registryImpl) GetAvailableTools() []schema.ToolDefinition {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	defs := make([]schema.ToolDefinition, 0, len(r.tools))
	for _, name := range names {
		defs = append(defs, r.tools[name].Definition())
	}

	return defs
}

func (r *registryImpl) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	tool, exists := r.tools[call.Name]
	if !exists {
		errMsg := fmt.Sprintf("Error: 系统中不存在名为 '%s' 的工具。", call.Name)
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     errMsg,
			IsError:    true,
		}
	}

	output, err := tool.Execute(ctx, call.Arguments)
	if err != nil {
		errMsg := fmt.Sprintf("Error executing %s: %v", call.Name, err)
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     errMsg,
			IsError:    true,
		}
	}

	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     output,
		IsError:    false,
	}
}
