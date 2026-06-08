package engine

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/yanking/go-tiny-claw/internal/provider"
	"github.com/yanking/go-tiny-claw/internal/reporter"
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

func (e *AgentEngine) Run(ctx context.Context, userPrompt string, reporter reporter.Reporter) error {
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
			if reporter != nil {
				reporter.OnThinking(ctx)
			}
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
		if responseMsg.Content != "" && reporter != nil {
			reporter.OnMessage(ctx, responseMsg.Content)
		}

		// 如果模型没有请求任何工具调用，说明它认为任务已经完成，跳出循环。
		if len(responseMsg.ToolCalls) == 0 {
			break
		}

		log.Printf("[engine] 模型请求调用 %d 个工具...\n", len(responseMsg.ToolCalls))
		observationMsgs := make([]schema.Message, len(responseMsg.ToolCalls))
		var wg sync.WaitGroup
		for idx, toolCall := range responseMsg.ToolCalls {
			wg.Add(1)
			go func(idx int, toolCall schema.ToolCall) {
				defer wg.Done()
				if reporter != nil {
					reporter.OnToolCall(ctx, toolCall.Name, string(toolCall.Arguments))
				}
				result := e.registry.Execute(ctx, toolCall)
				if reporter != nil {
					reporter.OnToolResult(ctx, toolCall.Name, result.Output, result.IsError)
				}

				observationMsgs[idx] = schema.Message{
					Role:       schema.RoleUser,
					Content:    result.Output,
					ToolCallID: toolCall.ID,
				}
			}(idx, toolCall)
		}
		wg.Wait()
		for _, obs := range observationMsgs {
			contextHistory = append(contextHistory, obs)
		}
	}

	return nil
}
