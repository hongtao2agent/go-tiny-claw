package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

type EditFileTool struct {
	workDir string
}

func NewEditFileTool(workDir string) *EditFileTool {
	return &EditFileTool{workDir: workDir}
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        t.Name(),
		Description: "对现有文件进行局部的字符串替换。这比重写整个文件更安全、更快速。请提供足够的 old_text 上下文以确保匹配的唯一性。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "要修改的文件路径",
				},
				"old_text": map[string]interface{}{
					"type":        "string",
					"description": "文件中原有的文本。必须包含足够的上下文（建议上下各多包含几行），以确保在文件中的唯一性。",
				},
				"new_text": map[string]interface{}{
					"type":        "string",
					"description": "要替换成的新文本",
				},
			},
			"required": []string{"path", "old_text", "new_text"},
		},
	}
}

type editFileArgs struct {
	Path    string `json:"path"`
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
}

func (t *EditFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	var input editFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	if strings.TrimSpace(input.Path) == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	if input.OldText == "" {
		return "", fmt.Errorf("old_text 不能为空")
	}

	fullPath, err := resolveWorkspacePath(t.workDir, input.Path)
	if err != nil {
		return "", err
	}

	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败，请确认路径是否正确: %w", err)
	}

	newContent, err := fuzzyReplace(string(contentBytes), input.OldText, input.NewText)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(fullPath, []byte(newContent), 0o644); err != nil {
		return "", fmt.Errorf("写回文件失败: %w", err)
	}

	return fmt.Sprintf("✅ 成功修改文件: %s", input.Path), nil
}

func fuzzyReplace(originalContent, oldText, newText string) (string, error) {
	count := strings.Count(originalContent, oldText)
	if count == 1 {
		return strings.Replace(originalContent, oldText, newText, 1), nil
	}
	if count > 1 {
		return "", fmt.Errorf("old_text 匹配到了 %d 处，请提供更多的上下文代码以确保唯一性", count)
	}

	normalizedContent := normalizeNewlines(originalContent)
	normalizedOld := normalizeNewlines(oldText)
	normalizedNew := normalizeNewlines(newText)

	count = strings.Count(normalizedContent, normalizedOld)
	if count == 1 {
		return strings.Replace(normalizedContent, normalizedOld, normalizedNew, 1), nil
	}
	if count > 1 {
		return "", fmt.Errorf("old_text 在换行归一化后匹配到了 %d 处，请提供更多上下文代码以确保唯一性", count)
	}

	trimmedOld := strings.TrimSpace(normalizedOld)
	if trimmedOld != "" {
		count = strings.Count(normalizedContent, trimmedOld)
		if count > 1 {
			return "", fmt.Errorf("old_text 去除首尾空白后匹配到了 %d 处，请提供更多上下文代码以确保唯一性", count)
		}
		if count == 1 && strings.Contains(trimmedOld, "\n") {
			return lineByLineReplace(normalizedContent, trimmedOld, normalizedNew)
		}
		if count == 1 {
			return strings.Replace(normalizedContent, trimmedOld, strings.TrimSpace(normalizedNew), 1), nil
		}
	}

	return lineByLineReplace(normalizedContent, normalizedOld, normalizedNew)
}

func lineByLineReplace(content, oldText, newText string) (string, error) {
	contentLines := strings.Split(content, "\n")
	oldLines := strings.Split(strings.TrimSpace(oldText), "\n")
	if len(oldLines) == 0 || len(contentLines) < len(oldLines) {
		return "", fmt.Errorf("找不到该代码片段")
	}

	for i := range oldLines {
		oldLines[i] = strings.TrimSpace(oldLines[i])
	}

	matchCount := 0
	matchStartIndex := -1
	matchEndIndex := -1

	for i := 0; i <= len(contentLines)-len(oldLines); i++ {
		isMatch := true
		for j := 0; j < len(oldLines); j++ {
			if strings.TrimSpace(contentLines[i+j]) != oldLines[j] {
				isMatch = false
				break
			}
		}

		if isMatch {
			matchCount++
			matchStartIndex = i
			matchEndIndex = i + len(oldLines)
		}
	}

	if matchCount == 0 {
		return "", fmt.Errorf("在文件中未找到 old_text，请大模型先调用 read_file 仔细确认文件内容和缩进")
	}
	if matchCount > 1 {
		return "", fmt.Errorf("模糊匹配到了 %d 处相似代码，请提供更多上下行代码以精确定位", matchCount)
	}

	replacementLines := strings.Split(strings.TrimSpace(newText), "\n")
	baseIndent := leadingWhitespace(contentLines[matchStartIndex])
	if baseIndent != "" {
		for i, line := range replacementLines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			replacementLines[i] = baseIndent + strings.TrimLeft(line, " \t")
		}
	}

	newContentLines := make([]string, 0, len(contentLines)-len(oldLines)+len(replacementLines))
	newContentLines = append(newContentLines, contentLines[:matchStartIndex]...)
	newContentLines = append(newContentLines, replacementLines...)
	newContentLines = append(newContentLines, contentLines[matchEndIndex:]...)

	return strings.Join(newContentLines, "\n"), nil
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func leadingWhitespace(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[:i]
		}
	}
	return s
}
