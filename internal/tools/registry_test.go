package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

type stubTool struct {
	name string
	err  error
}

func (t stubTool) Name() string {
	return t.name
}

func (t stubTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{Name: t.name}
}

func (t stubTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if t.err != nil {
		return "", t.err
	}
	return string(args), nil
}

func TestRegistryExecuteRoutesKnownTool(t *testing.T) {
	registry := NewRegistry()
	registry.Register(stubTool{name: "echo"})

	result := registry.Execute(context.Background(), schema.ToolCall{
		ID:        "call_1",
		Name:      "echo",
		Arguments: json.RawMessage(`{"ok":true}`),
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Output)
	}
	if result.ToolCallID != "call_1" {
		t.Fatalf("unexpected tool call id: %s", result.ToolCallID)
	}
	if result.Output != `{"ok":true}` {
		t.Fatalf("unexpected output: %s", result.Output)
	}
}

func TestRegistryExecuteReturnsToolError(t *testing.T) {
	registry := NewRegistry()
	registry.Register(stubTool{name: "fail", err: errors.New("boom")})

	result := registry.Execute(context.Background(), schema.ToolCall{
		ID:   "call_1",
		Name: "fail",
	})

	if !result.IsError {
		t.Fatalf("expected error result")
	}
}

func TestRegistryExecuteRejectsUnknownTool(t *testing.T) {
	registry := NewRegistry()

	result := registry.Execute(context.Background(), schema.ToolCall{
		ID:   "call_1",
		Name: "missing",
	})

	if !result.IsError {
		t.Fatalf("expected error result")
	}
}
