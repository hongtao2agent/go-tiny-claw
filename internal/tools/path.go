package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

func resolveWorkspacePath(workDir string, path string) (string, error) {
	workDirAbs, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("解析工作区路径失败: %w", err)
	}

	fullPath := filepath.Join(workDirAbs, path)
	fullPathAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("解析文件路径失败: %w", err)
	}

	rel, err := filepath.Rel(workDirAbs, fullPathAbs)
	if err != nil {
		return "", fmt.Errorf("校验文件路径失败: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("拒绝访问工作区外的路径: %s", path)
	}

	return fullPathAbs, nil
}
