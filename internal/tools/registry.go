package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/yanking/go-tiny-claw/internal/schema"
)

// Registry 定义了工具的注册与分发执行接口
type Registry interface {
	// Register 挂载一个新的工具到系统中
	Register(tool BaseTool)

	// GetAvailableTools 返回当前系统挂载的所有可用工具的 Schema
	GetAvailableTools() []schema.ToolDefinition

	// Execute 实际执行模型请求的工具，并返回结果
	Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult
}

type BaseTool interface {
	Name() string
	Definition() schema.ToolDefinition
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

type registryImpl struct {
	tools map[string]BaseTool
}

func NewRegistry() Registry {
	return &registryImpl{
		tools: make(map[string]BaseTool),
	}
}

func (r *registryImpl) Register(tool BaseTool) {
	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		log.Printf("[tool] 工具 '%s' 已被注册，将被覆盖\n", name)
	}
	r.tools[name] = tool
	log.Printf("[tool] 成功挂载工具： %s\n", name)
}

func (r *registryImpl) GetAvailableTools() []schema.ToolDefinition {
	var def []schema.ToolDefinition
	for _, tool := range r.tools {
		def = append(def, tool.Definition())
	}
	return def
}

func (r *registryImpl) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	tool, exists := r.tools[call.Name]
	if !exists {
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     fmt.Sprintf("系统中不存在名为 ‘%s’ 的工具", call.Name),
			IsError:    true,
		}
	}

	output, err := tool.Execute(ctx, call.Arguments)
	if err != nil {
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     fmt.Sprintf("error executing %s: %v", call.Name, err),
			IsError:    true,
		}
	}

	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     output,
		IsError:    false,
	}
}
