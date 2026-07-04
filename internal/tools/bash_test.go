package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestBashToolRunsCommandInWorkspace(t *testing.T) {
	dir := t.TempDir()
	tool := NewBashTool(dir)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"command":"pwd"}`))
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if strings.TrimSpace(output) != dir {
		t.Fatalf("expected command to run in workspace %q, got %q", dir, strings.TrimSpace(output))
	}
}

func TestBashToolReturnsNoOutputMessage(t *testing.T) {
	dir := t.TempDir()
	tool := NewBashTool(dir)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"command":"true"}`))
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if output != "命令执行成功，无终端输出。" {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestBashToolReturnsFailureAsOutput(t *testing.T) {
	dir := t.TempDir()
	tool := NewBashTool(dir)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"command":"echo nope && exit 7"}`))
	if err != nil {
		t.Fatalf("expected command failures to be returned as output: %v", err)
	}
	if !strings.Contains(output, "执行报错") || !strings.Contains(output, "nope") {
		t.Fatalf("unexpected output: %s", output)
	}
}
