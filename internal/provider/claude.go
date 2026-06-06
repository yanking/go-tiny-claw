package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/yanking/go-tiny-claw/internal/schema"
)

type ClaudeProvider struct {
	client anthropic.Client
	model  string
}

func NewClaudeProvider(apiKey string, baseURL string, model string) *ClaudeProvider {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := anthropic.NewClient(opts...)
	return &ClaudeProvider{client: client, model: model}
}

func (c ClaudeProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	var systemBlocks []anthropic.TextBlockParam
	var messageParams []anthropic.MessageParam

	for _, msg := range msgs {
		switch {
		case msg.ToolCallID != "":
			// 工具执行结果，作为 user 消息中的 tool_result 块
			messageParams = append(messageParams, anthropic.NewUserMessage(
				anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
			))

		case msg.Role == schema.RoleSystem:
			systemBlocks = append(systemBlocks, anthropic.TextBlockParam{
				Text: msg.Content,
			})

		case msg.Role == schema.RoleUser:
			messageParams = append(messageParams, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))

		case msg.Role == schema.RoleAssistant:
			var blocks []anthropic.ContentBlockParamUnion
			if msg.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
			}
			for _, tc := range msg.ToolCalls {
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, tc.Arguments, tc.Name))
			}
			if len(blocks) == 0 {
				blocks = append(blocks, anthropic.NewTextBlock(""))
			}
			messageParams = append(messageParams, anthropic.NewAssistantMessage(blocks...))
		}
	}

	// 转换工具定义
	toolParams := make([]anthropic.ToolUnionParam, 0, len(availableTools))
	for _, tool := range availableTools {
		schemaMap, ok := tool.InputSchema.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("tool %q: InputSchema 必须是 map[string]any 类型", tool.Name)
		}

		tp := anthropic.ToolParam{
			Name:        tool.Name,
			Description: param.NewOpt(tool.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: schemaMap["properties"],
				Required:   toStringSlice(schemaMap["required"]),
			},
		}
		toolParams = append(toolParams, anthropic.ToolUnionParam{
			OfTool: &tp,
		})
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 4096,
		Messages:  messageParams,
	}

	if len(systemBlocks) > 0 {
		params.System = systemBlocks
	}

	if len(toolParams) > 0 {
		params.Tools = toolParams
	}

	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("claude api 调用失败: %w", err)
	}

	// 将响应转换回 schema.Message
	result := &schema.Message{
		Role: schema.RoleAssistant,
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			result.Content += block.Text
		case "tool_use":
			result.ToolCalls = append(result.ToolCalls, schema.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: json.RawMessage(block.Input),
			})
		}
	}

	return result, nil
}

var _ LLMProvider = (*ClaudeProvider)(nil)

// toStringSlice 将 interface{} 转换为 []string
func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		result := make([]string, len(s))
		for i, item := range s {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	default:
		return nil
	}
}
