package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

type ReadFileTool struct {
	workDir string
}

func NewReadFileTool(workDir string) *ReadFileTool {
	return &ReadFileTool{workDir: workDir}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        t.Name(),
		Description: "读取指定路径的文件内容。请提供相对工作区的路径。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "要读取的文件路径，如 cmd/claw/main.go",
				},
			},
			"required": []string{"path"},
		},
	}
}

type readFileArgs struct {
	Path string `json:"path"`
}

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	var input readFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	if strings.TrimSpace(input.Path) == "" {
		return "", fmt.Errorf("path 不能为空")
	}

	fullPath, err := t.resolvePath(input.Path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("读取文件信息失败: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("目标路径是目录，不是文件: %s", input.Path)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("读取文件内容失败: %w", err)
	}

	const maxLen = 8000
	if len(content) > maxLen {
		truncatedMsg := fmt.Sprintf("%s\n\n...[由于内容过长，已被系统截断至前 %d 字节]...", string(content[:maxLen]), maxLen)
		return truncatedMsg, nil
	}

	return string(content), nil
}

func (t *ReadFileTool) resolvePath(path string) (string, error) {
	return resolveWorkspacePath(t.workDir, path)
}
