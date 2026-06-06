package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yanking/go-tiny-claw/internal/schema"
)

type ReadFileTool struct {
	workDir string
}

func NewReadFileTool(workDir string) *ReadFileTool {
	return &ReadFileTool{
		workDir: workDir,
	}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        t.Name(),
		Description: "Read a file from the workspace.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path relative to workspace",
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
	var input readFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	fullPath, err := safePath(t.workDir, input.Path)
	if err != nil {
		return "", err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	const maxLen = 8000
	if len(content) > maxLen {
		return string(content[:maxLen]) + "\n...(truncated)", nil
	}

	return string(content), nil
}

// safePath 解析并校验路径，防止 .. 越界访问工作区外的文件
func safePath(workDir, relPath string) (string, error) {
	abs := filepath.Clean(filepath.Join(workDir, relPath))
	base := filepath.Clean(workDir)
	rel, err := filepath.Rel(base, abs)
	if err != nil {
		return "", fmt.Errorf("path escapes workspace: %s", relPath)
	}
	if rel == ".." || len(rel) >= 3 && rel[:3] == "../" {
		return "", fmt.Errorf("path escapes workspace: %s", relPath)
	}
	return abs, nil
}
