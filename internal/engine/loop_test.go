package engine

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/yourname/go-tiny-claw/internal/schema"
	"github.com/yourname/go-tiny-claw/internal/tools"
)

type testProvider struct{}

func (p *testProvider) Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	return &schema.Message{Role: schema.RoleAssistant}, nil
}

type delayedRegistry struct {
	delays map[string]time.Duration

	mu       sync.Mutex
	started  int
	maxLive  int
	live     int
	finished []string
}

func (r *delayedRegistry) Register(tool tools.BaseTool) {}

func (r *delayedRegistry) GetAvailableTools() []schema.ToolDefinition {
	return nil
}

func (r *delayedRegistry) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	r.mu.Lock()
	r.started++
	r.live++
	if r.live > r.maxLive {
		r.maxLive = r.live
	}
	r.mu.Unlock()

	time.Sleep(r.delays[call.ID])

	r.mu.Lock()
	r.live--
	r.finished = append(r.finished, call.ID)
	r.mu.Unlock()

	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     "result-" + call.ID,
		IsError:    false,
	}
}

func TestExecuteToolCallsRunsConcurrentlyAndPreservesOrder(t *testing.T) {
	registry := &delayedRegistry{
		delays: map[string]time.Duration{
			"first":  50 * time.Millisecond,
			"second": 10 * time.Millisecond,
			"third":  20 * time.Millisecond,
		},
	}
	engine := NewAgentEngine(&testProvider{}, registry, t.TempDir(), false)

	start := time.Now()
	observations := engine.executeToolCalls(context.Background(), []schema.ToolCall{
		{ID: "first", Name: "stub", Arguments: json.RawMessage(`{"n":1}`)},
		{ID: "second", Name: "stub", Arguments: json.RawMessage(`{"n":2}`)},
		{ID: "third", Name: "stub", Arguments: json.RawMessage(`{"n":3}`)},
	}, nil)
	elapsed := time.Since(start)

	if elapsed >= 80*time.Millisecond {
		t.Fatalf("expected concurrent execution, took %s", elapsed)
	}
	if registry.maxLive < 2 {
		t.Fatalf("expected overlapping executions, max live was %d", registry.maxLive)
	}

	expectedOrder := []string{"first", "second", "third"}
	for i, expectedID := range expectedOrder {
		if observations[i].ToolCallID != expectedID {
			t.Fatalf("observation %d id = %q, want %q", i, observations[i].ToolCallID, expectedID)
		}
		if observations[i].Content != "result-"+expectedID {
			t.Fatalf("observation %d content = %q", i, observations[i].Content)
		}
	}
}
