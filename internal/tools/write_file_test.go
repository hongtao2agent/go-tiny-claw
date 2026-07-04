package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileToolCreatesParentDirectories(t *testing.T) {
	dir := t.TempDir()
	tool := NewWriteFileTool(dir)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"nested/hello.txt","content":"hello tiny claw"}`))
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if !strings.Contains(output, "nested/hello.txt") {
		t.Fatalf("unexpected output: %s", output)
	}

	content, err := os.ReadFile(filepath.Join(dir, "nested", "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello tiny claw" {
		t.Fatalf("unexpected file content: %s", content)
	}
}

func TestWriteFileToolRejectsOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	tool := NewWriteFileTool(dir)

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"../outside.txt","content":"secret"}`))
	if err == nil {
		t.Fatalf("expected workspace boundary error")
	}
}
