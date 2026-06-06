// cmd/claw/main.go
package main

import (
	"context"
	"log"
	"os"

	"github.com/yanking/go-tiny-claw/internal/engine"
	"github.com/yanking/go-tiny-claw/internal/provider"
	"github.com/yanking/go-tiny-claw/internal/schema"
)

// 伪造的工具注册表 (用于测试 Provider 的工具提取能力)
type mockRegistry struct{}

func (m *mockRegistry) GetAvailableTools() []schema.ToolDefinition {
	return []schema.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "获取指定城市的当前天气情况。",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"city": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"city"},
			},
		},
	}
}

func (m *mockRegistry) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	log.Printf("  -> [Mock 工具执行] 获取 %s 的天气中...\n", call.Name)
	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     "API 返回：今天是晴天，气温 25 度。",
		IsError:    false,
	}
}

func main() {
	apiKey := os.Getenv("CLAW_API_KEY")
	baseURL := os.Getenv("CLAW_BASE_URL")
	model := os.Getenv("CLAW_MODEL")
	if model == "" {
		model = "glm-5.1"
	}

	if apiKey == "" {
		log.Fatal("请设置环境变量 CLAW_API_KEY")
	}

	workDir, _ := os.Getwd()
	llmProvider := provider.NewOpenAIProvider(apiKey, baseURL, model)

	registry := &mockRegistry{}
	eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

	prompt := "我想去北京跑步，帮我查查天气适合吗？"

	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}
