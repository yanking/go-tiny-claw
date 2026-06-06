package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
	"github.com/yanking/go-tiny-claw/internal/schema"
)

type OpenAIProvider struct {
	client openai.Client
	model  string
}

func NewOpenAIProvider(apiKey string, baseURL string, model string) *OpenAIProvider {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := openai.NewClient(opts...)
	return &OpenAIProvider{client: client, model: model}
}

func (o OpenAIProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	var messageParams []openai.ChatCompletionMessageParamUnion

	for _, msg := range msgs {
		switch {
		case msg.ToolCallID != "":
			// 工具执行结果
			messageParams = append(messageParams, openai.ToolMessage(msg.Content, msg.ToolCallID))

		case msg.Role == schema.RoleSystem:
			messageParams = append(messageParams, openai.SystemMessage(msg.Content))

		case msg.Role == schema.RoleUser:
			messageParams = append(messageParams, openai.UserMessage(msg.Content))

		case msg.Role == schema.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				// 带工具调用的 assistant 消息
				toolCallParams := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(msg.ToolCalls))
				for _, tc := range msg.ToolCalls {
					toolCallParams = append(toolCallParams, openai.ChatCompletionMessageToolCallUnionParam{
						OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
							ID: tc.ID,
							Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
								Arguments: string(tc.Arguments),
								Name:      tc.Name,
							},
						},
					})
				}

				asst := openai.ChatCompletionAssistantMessageParam{
					ToolCalls: toolCallParams,
				}
				if msg.Content != "" {
					asst.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(msg.Content),
					}
				}
				messageParams = append(messageParams, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &asst,
				})
			} else {
				messageParams = append(messageParams, openai.AssistantMessage(msg.Content))
			}
		}
	}

	// 转换工具定义
	toolParams := make([]openai.ChatCompletionToolUnionParam, 0, len(availableTools))
	for _, tool := range availableTools {
		params := shared.FunctionParameters{}
		if schemaMap, ok := tool.InputSchema.(map[string]any); ok {
			params = shared.FunctionParameters(schemaMap)
		}

		toolParams = append(toolParams, openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        tool.Name,
					Description: openai.String(tool.Description),
					Parameters:  params,
				},
			},
		})
	}

	reqParams := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(o.model),
		Messages: messageParams,
	}

	if len(toolParams) > 0 {
		reqParams.Tools = toolParams
	}

	resp, err := o.client.Chat.Completions.New(ctx, reqParams)
	if err != nil {
		return nil, fmt.Errorf("openai api 调用失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai api 返回空响应")
	}

	choice := resp.Choices[0]
	result := &schema.Message{
		Role:    schema.RoleAssistant,
		Content: choice.Message.Content,
	}

	// 转换工具调用
	for _, tc := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, schema.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: json.RawMessage(tc.Function.Arguments),
		})
	}

	return result, nil
}

var _ LLMProvider = (*OpenAIProvider)(nil)
