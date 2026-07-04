package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFuzzyReplaceExactMatch(t *testing.T) {
	output, err := fuzzyReplace("hello world", "world", "tiny claw")
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if output != "hello tiny claw" {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestFuzzyReplaceRejectsMultipleExactMatches(t *testing.T) {
	_, err := fuzzyReplace("same\nsame\n", "same", "new")
	if err == nil {
		t.Fatalf("expected multiple match error")
	}
}

func TestFuzzyReplaceNormalizesNewlines(t *testing.T) {
	output, err := fuzzyReplace("a\r\nb\r\nc", "a\nb", "x\ny")
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if output != "x\ny\nc" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestFuzzyReplaceTrimsOuterWhitespace(t *testing.T) {
	output, err := fuzzyReplace("before\nold\ntext\nafter", "\n\nold\ntext\n\n", "new")
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if output != "before\nnew\nafter" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestFuzzyReplaceIgnoresLineIndentation(t *testing.T) {
	input := "func main() {\n    if true {\n        fmt.Println(\"ok\")\n    }\n}\n"
	oldText := "if true {\nfmt.Println(\"ok\")\n}"
	newText := "if user == nil {\nfmt.Println(\"Forbidden!\")\nreturn\n}"

	output, err := fuzzyReplace(input, oldText, newText)
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}

	expected := "func main() {\n    if user == nil {\n    fmt.Println(\"Forbidden!\")\n    return\n    }\n}\n"
	if output != expected {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestFuzzyReplaceRejectsMultipleFuzzyMatches(t *testing.T) {
	input := "if true {\n    x()\n}\nif true {\n    x()\n}\n"
	_, err := fuzzyReplace(input, "if true {\nx()\n}", "y()")
	if err == nil {
		t.Fatalf("expected multiple fuzzy match error")
	}
}

func TestEditFileToolEditsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.go")
	if err := os.WriteFile(path, []byte("package main\n\nfunc main() {\n    if true {\n        println(\"open\")\n    }\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tool := NewEditFileTool(dir)
	args := map[string]string{
		"path":     "server.go",
		"old_text": "if true {\nprintln(\"open\")\n}",
		"new_text": "if user == nil {\nprintln(\"forbidden\")\nreturn\n}",
	}
	rawArgs, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}

	output, err := tool.Execute(context.Background(), rawArgs)
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if !strings.Contains(output, "成功修改文件") {
		t.Fatalf("unexpected output: %s", output)
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "println(\"forbidden\")") {
		t.Fatalf("file was not edited:\n%s", updated)
	}
}

func TestEditFileToolRejectsOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	tool := NewEditFileTool(dir)

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"../outside.go","old_text":"a","new_text":"b"}`))
	if err == nil {
		t.Fatalf("expected workspace boundary error")
	}
}
