package engine

import (
	"context"
	"fmt"
	"log"

	"github.com/yanking/go-tiny-claw/internal/provider"
	"github.com/yanking/go-tiny-claw/internal/schema"
	"github.com/yanking/go-tiny-claw/internal/tools"
)

// AgentEngine 是微型 OS 的核心驱动
type AgentEngine struct {
	provider       provider.LLMProvider
	registry       tools.Registry
	WorkDir        string
	EnableThinking bool
}

func NewAgentEngine(provider provider.LLMProvider, registry tools.Registry, workDir string, enableThinking bool) *AgentEngine {
	return &AgentEngine{
		provider:       provider,
		registry:       registry,
		WorkDir:        workDir,
		EnableThinking: enableThinking,
	}
}

func (e *AgentEngine) Run(ctx context.Context, userPrompt string) error {
	log.Printf("[engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)
	log.Printf("[engine] Thinking Phase %v\n", e.EnableThinking)
	contextHistory := []schema.Message{
		{
			Role:    schema.RoleSystem,
			Content: "You are go-tiny-claw, an expert coding assistant. You have full access to tools in the workspace.",
		},
		{
			Role:    schema.RoleUser,
			Content: userPrompt,
		},
	}

	turn := 0
	for {
		turn++
		log.Printf("[turn %d] 开始======\n", turn)
		availableTools := e.registry.GetAvailableTools()

		if e.EnableThinking {
			log.Printf("[engine] thinking...")
			thinkResp, err := e.provider.Generate(ctx, contextHistory, nil)
			if err != nil {
				return fmt.Errorf("[engine] thinking : %w", err)
			}
			if thinkResp.Content != "" {
				fmt.Printf("🧠  [thinking]: %s\n", thinkResp.Content)
				contextHistory = append(contextHistory, *thinkResp)
			}
		}

		responseMsg, err := e.provider.Generate(ctx, contextHistory, availableTools)
		if err != nil {
			return fmt.Errorf("[engine] 生成失败: %w", err)
		}

		contextHistory = append(contextHistory, *responseMsg)
		if responseMsg.Content != "" {
			fmt.Printf("🧠: %s\n", responseMsg.Content)
		}

		// 如果模型没有请求任何工具调用，说明它认为任务已经完成，跳出循环。
		if len(responseMsg.ToolCalls) == 0 {
			log.Println("[engine] 任务完成，退出循环")
			break
		}

		log.Printf("[engine] 模型请求调用 %d 个工具...\n", len(responseMsg.ToolCalls))
		for _, toolCall := range responseMsg.ToolCalls {
			log.Printf("  -> ⚙ 执行工具: %s 参数： %s\n", toolCall.Name, string(toolCall.Arguments))
			result := e.registry.Execute(ctx, toolCall)
			if result.IsError {
				log.Printf("  -> ❌ 工具执行报错: %s\n", result.Output)
			} else {
				log.Printf("  -> ✅ 工具执行成功: 返回 %d 字节\n", len(result.Output))
			}

			observationMsg := schema.Message{
				Role:       schema.RoleUser,
				Content:    result.Output,
				ToolCallID: toolCall.ID,
			}
			contextHistory = append(contextHistory, observationMsg)
		}
	}

	return nil
}
