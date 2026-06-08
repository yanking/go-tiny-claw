package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// TelegramBot 封装 Telegram Bot API 的 HTTP 调用
type TelegramBot struct {
	token   string
	chatID  string
	client  *http.Client
	apiBase string // 可被测试覆盖，默认 "https://api.telegram.org"
}

// NewTelegramBot 创建一个 TelegramBot 实例
func NewTelegramBot(token, chatID string) *TelegramBot {
	return &TelegramBot{
		token:   token,
		chatID:  chatID,
		client:  &http.Client{},
		apiBase: "https://api.telegram.org",
	}
}

// telegramSendMessageRequest 对应 Telegram sendMessage API 的请求体
type telegramSendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// sendText 向目标聊天发送纯文本消息（MarkdownV2 格式）
func (b *TelegramBot) sendText(text string) error {
	url := fmt.Sprintf("%s/bot%s/sendMessage", b.apiBase, b.token)
	body := telegramSendMessageRequest{
		ChatID:    b.chatID,
		Text:      text,
		ParseMode: "MarkdownV2",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	resp, err := b.client.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram api status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

// markdownV2Specials 匹配需要转义的字符
var markdownV2Specials = regexp.MustCompile(`([_*\[\]()~` + "`" + `>#\+\-=|{}.!\\])`)

// escapeMarkdownV2 将文本中的 MarkdownV2 特殊字符进行转义
func escapeMarkdownV2(text string) string {
	return markdownV2Specials.ReplaceAllString(text, `\$1`)
}

// truncate 将文本截断到指定长度，超出时追加省略号
func truncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "\n...(truncated)"
}

// OnThinking 发送思考中状态消息
func (b *TelegramBot) OnThinking(ctx context.Context) {
	_ = ctx
	text := "🧠 思考中\\.\\.\\."
	if err := b.sendText(text); err != nil {
		log.Printf("[telegram] OnThinking 发送失败: %v", err)
	}
}

// OnToolCall 发送工具调用消息
func (b *TelegramBot) OnToolCall(ctx context.Context, toolName string, args string) {
	_ = ctx
	escaped := escapeMarkdownV2(truncate(args, 200))
	text := fmt.Sprintf("⚙ *调用工具:* `%s`\n```\n%s\n```", escapeMarkdownV2(toolName), escaped)
	if err := b.sendText(text); err != nil {
		log.Printf("[telegram] OnToolCall 发送失败: %v", err)
	}
}

// OnToolResult 发送工具执行结果消息
func (b *TelegramBot) OnToolResult(ctx context.Context, toolName string, result string, isError bool) {
	_ = ctx
	icon := "✅"
	if isError {
		icon = "❌"
	}
	escaped := escapeMarkdownV2(truncate(result, 1000))
	text := fmt.Sprintf("%s *%s 结果:*\n```\n%s\n```", icon, escapeMarkdownV2(toolName), escaped)
	if err := b.sendText(text); err != nil {
		log.Printf("[telegram] OnToolResult 发送失败: %v", err)
	}
}

// OnMessage 发送普通消息
func (b *TelegramBot) OnMessage(ctx context.Context, content string) {
	_ = ctx
	text := "💬 " + escapeMarkdownV2(truncate(content, 4000))
	if err := b.sendText(text); err != nil {
		log.Printf("[telegram] OnMessage 发送失败: %v", err)
	}
}
