package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileToolReadsWorkspaceFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello tiny claw"), 0o600); err != nil {
		t.Fatal(err)
	}

	tool := NewReadFileTool(dir)
	output, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"hello.txt"}`))
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if output != "hello tiny claw" {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestReadFileToolRejectsOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	parentFile := filepath.Join(filepath.Dir(dir), "outside.txt")
	if err := os.WriteFile(parentFile, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(parentFile)
	})

	tool := NewReadFileTool(dir)
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"../outside.txt"}`))
	if err == nil {
		t.Fatalf("expected workspace boundary error")
	}
}

func TestReadFileToolTruncatesLargeFile(t *testing.T) {
	dir := t.TempDir()
	content := strings.Repeat("a", 9000)
	if err := os.WriteFile(filepath.Join(dir, "large.txt"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	tool := NewReadFileTool(dir)
	output, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"large.txt"}`))
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if !strings.Contains(output, "已被系统截断至前 8000 字节") {
		t.Fatalf("expected truncation marker, got: %s", output)
	}
}
