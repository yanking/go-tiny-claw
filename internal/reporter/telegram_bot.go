package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
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
		return fmt.Errorf("telegram api returned status %d", resp.StatusCode)
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
