package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/yanking/go-tiny-claw/internal/schema"
)

type EditFileTool struct {
	workDir string
}

func NewEditFileTool(workDir string) *EditFileTool {
	return &EditFileTool{
		workDir: workDir,
	}
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name: t.Name(),
		Description: "Find and replace text in a file. Whitespace-tolerant: matches even if indentation or trailing whitespace differs. " +
			"Fails if old_string not found or appears more than once, unless replace_all is set.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path relative to workspace",
				},
				"old_string": map[string]interface{}{
					"type":        "string",
					"description": "Text to find in the file",
				},
				"new_string": map[string]interface{}{
					"type":        "string",
					"description": "Text to replace old_string with",
				},
				"replace_all": map[string]interface{}{
					"type":        "boolean",
					"description": "Replace all occurrences. Default false.",
				},
			},
			"required": []string{"path", "old_string", "new_string"},
		},
	}
}

type editFileArgs struct {
	Path       string `json:"path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all"`
}

func (t *EditFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var input editFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	fullPath, err := safePath(t.workDir, input.Path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	content := string(data)

	// 1. 精确匹配
	count := strings.Count(content, input.OldString)

	// 2. 精确失败后，按行模糊匹配（忽略行首缩进和行尾空白差异）
	if count == 0 {
		count = fuzzyLineCount(content, input.OldString)
	}

	switch {
	case count == 0:
		return "", fmt.Errorf("old_string not found in %s", input.Path)
	case count > 1 && !input.ReplaceAll:
		return "", fmt.Errorf("old_string matched %d times in %s; provide more context or set replace_all", count, input.Path)
	}

	// 执行替换
	if strings.Count(content, input.OldString) > 0 {
		if input.ReplaceAll {
			content = strings.ReplaceAll(content, input.OldString, input.NewString)
		} else {
			content = strings.Replace(content, input.OldString, input.NewString, 1)
		}
	} else {
		content = fuzzyReplace(content, input.OldString, input.NewString, input.ReplaceAll)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fmt.Sprintf("Replaced %d match(es) in %s", count, input.Path), nil
}

// fuzzyLineCount 统计模糊匹配次数（每行忽略行首缩进和行尾空白）
func fuzzyLineCount(content, old string) int {
	contentLines := strings.Split(content, "\n")
	oldNorm := normalizeLines(old)
	n := len(oldNorm)
	if n == 0 {
		return 0
	}

	count := 0
	for i := 0; i <= len(contentLines)-n; i++ {
		if matchNormBlock(contentLines, i, oldNorm) {
			count++
		}
	}
	return count
}

// fuzzyReplace 模糊替换
func fuzzyReplace(content, old, newStr string, replaceAll bool) string {
	contentLines := strings.Split(content, "\n")
	oldNorm := normalizeLines(old)
	n := len(oldNorm)
	newLines := strings.Split(newStr, "\n")

	result := make([]string, 0, len(contentLines))
	replaced := 0

	for i := 0; i < len(contentLines); {
		if (!replaceAll && replaced > 0) || i > len(contentLines)-n || !matchNormBlock(contentLines, i, oldNorm) {
			result = append(result, contentLines[i])
			i++
			continue
		}
		result = append(result, newLines...)
		i += n
		replaced++
	}

	return strings.Join(result, "\n")
}

// matchNormBlock 检查 contentLines 从 start 开始是否模糊匹配 oldNorm
func matchNormBlock(contentLines []string, start int, oldNorm []string) bool {
	for j, ol := range oldNorm {
		if normalizeLine(contentLines[start+j]) != ol {
			return false
		}
	}
	return true
}

// normalizeLines 将多行文本每行标准化
func normalizeLines(s string) []string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = normalizeLine(l)
	}
	return lines
}

// normalizeLine 去除行首缩进和行尾空白
func normalizeLine(line string) string {
	return strings.TrimSpace(line)
}
