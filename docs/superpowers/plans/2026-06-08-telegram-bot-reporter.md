# Telegram Bot Reporter 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 `reporter.Reporter` 接口，将 Agent 运行过程推送到 Telegram 聊天。

**Architecture:** 标准库 `net/http` 直接调用 Telegram Bot API `sendMessage`，MarkdownV2 格式化，内容截断保护，发送失败仅 log 不中断主流程。

**Tech Stack:** Go 1.26, 标准库 net/http, net/url, encoding/json, strconv

**Design spec:** `docs/superpowers/specs/2026-06-08-telegram-bot-reporter-design.md`

---

## 文件结构

| 文件 | 职责 |
|------|------|
| `internal/reporter/telegram_bot.go` | TelegramBot 结构体及 Reporter 接口实现 |
| `internal/reporter/telegram_bot_test.go` | 单元测试，用 httptest 模拟 Telegram API |

---

### Task 1: 核心结构与发送函数

**Files:**
- Create: `internal/reporter/telegram_bot.go`
- Create: `internal/reporter/telegram_bot_test.go`

- [ ] **Step 1: 编写测试 — 构造函数与 sendMessage**

在 `internal/reporter/telegram_bot_test.go` 中：

```go
package reporter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewTelegramBot(t *testing.T) {
	bot := NewTelegramBot("123:ABC", "999")
	if bot.token != "123:ABC" {
		t.Errorf("token = %q, want %q", bot.token, "123:ABC")
	}
	if bot.chatID != "999" {
		t.Errorf("chatID = %q, want %q", bot.chatID, "999")
	}
	if bot.client == nil {
		t.Error("client should not be nil")
	}
}

// captureRequest 启动一个 mock Telegram API 服务器，返回收到的请求体
func captureRequest(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) (*TelegramBot, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(ts.Close)
	bot := NewTelegramBot("123:ABC", "999")
	bot.apiBase = ts.URL
	bot.client = ts.Client()
	return bot, ts
}

func parseSendMessageBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func TestSendTextMessage(t *testing.T) {
	var gotBody map[string]interface{}
	bot, _ := captureRequest(t, func(w http.ResponseWriter, r *http.Request) {
		gotBody = parseSendMessageBody(t, r)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})

	err := bot.sendText("hello world")
	if err != nil {
		t.Fatalf("sendText: %v", err)
	}
	if gotBody["chat_id"] != "999" {
		t.Errorf("chat_id = %v, want 999", gotBody["chat_id"])
	}
	if gotBody["text"] != "hello world" {
		t.Errorf("text = %v, want hello world", gotBody["text"])
	}
	if gotBody["parse_mode"] != "MarkdownV2" {
		t.Errorf("parse_mode = %v, want MarkdownV2", gotBody["parse_mode"])
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/reporter/ -run "TestNewTelegramBot|TestSendTextMessage" -v`
Expected: 编译失败，`NewTelegramBot` 和 `sendText` 未定义

- [ ] **Step 3: 实现核心结构**

在 `internal/reporter/telegram_bot.go` 中：

```go
package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type TelegramBot struct {
	token   string
	chatID  string
	client  *http.Client
	apiBase string // 可被测试覆盖，默认 "https://api.telegram.org"
}

func NewTelegramBot(token, chatID string) *TelegramBot {
	return &TelegramBot{
		token:   token,
		chatID:  chatID,
		client:  &http.Client{},
		apiBase: "https://api.telegram.org",
	}
}

type telegramSendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

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
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/reporter/ -run "TestNewTelegramBot|TestSendTextMessage" -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/reporter/telegram_bot.go internal/reporter/telegram_bot_test.go
git commit -m "feat(reporter): 添加 TelegramBot 核心结构与 sendText"
```

---

### Task 2: MarkdownV2 转义与内容截断

**Files:**
- Modify: `internal/reporter/telegram_bot.go`
- Modify: `internal/reporter/telegram_bot_test.go`

- [ ] **Step 1: 编写测试 — escapeMarkdownV2 与 truncate**

在 `telegram_bot_test.go` 末尾追加：

```go
func TestEscapeMarkdownV2(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"hello", "hello"},
		{"hello_world", "hello\\_world"},
		{"a*b[c]d", "a\\*b\\[c\\]d"},
		{"a(b)c`d", "a\\(b\\)c\\`d"},
		{"a~b>c#d", "a\\~b\\>c\\#d"},
		{"a+b-d=e", "a\\+b\\-d\\=e"},
		{"a|b{c}d", "a\\|b\\{c\\}d"},
		{"a.b!d", "a\\.b\\!d"},
		{`a\b`, `a\\\\b`},
	}
	for _, tt := range tests {
		got := escapeMarkdownV2(tt.input)
		if got != tt.expect {
			t.Errorf("escapeMarkdownV2(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestTruncate(t *testing.T) {
	// 短内容不截断
	if got := truncate("abc", 10); got != "abc" {
		t.Errorf("truncate short = %q, want %q", got, "abc")
	}
	// 长内容截断并追加省略号
	long := string(make([]byte, 100))
	for i := range long {
		long[i] = 'x'
	}
	got := truncate(long, 50)
	if len(got) > 55 { // 50 + "\n...(truncated)" 长度
		t.Errorf("truncate long len = %d, too long", len(got))
	}
	if len(got) < 50 {
		t.Errorf("truncate long len = %d, should keep first 50", len(got))
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/reporter/ -run "TestEscapeMarkdownV2|TestTruncate" -v`
Expected: 编译失败

- [ ] **Step 3: 实现转义与截断函数**

在 `telegram_bot.go` 末尾追加：

```go
import (
	"regexp"
)

// markdownV2Specials 匹配需要转义的字符
var markdownV2Specials = regexp.MustCompile(`([_*\[\]()~` + "`" + `>#\+\-=|{}.!\\])`)

func escapeMarkdownV2(text string) string {
	return markdownV2Specials.ReplaceAllString(text, `\$1`)
}

func truncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "\n...(truncated)"
}
```

注意：在文件顶部的 import 块中添加 `"regexp"`。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/reporter/ -run "TestEscapeMarkdownV2|TestTruncate" -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/reporter/telegram_bot.go internal/reporter/telegram_bot_test.go
git commit -m "feat(reporter): 添加 MarkdownV2 转义与内容截断"
```

---

### Task 3: 实现 Reporter 接口的四个方法

**Files:**
- Modify: `internal/reporter/telegram_bot.go`
- Modify: `internal/reporter/telegram_bot_test.go`

- [ ] **Step 1: 编写测试 — 四个 Reporter 方法**

在 `telegram_bot_test.go` 末尾追加：

```go
func TestOnThinking(t *testing.T) {
	var gotText string
	bot, _ := captureRequest(t, func(w http.ResponseWriter, r *http.Request) {
		body := parseSendMessageBody(t, r)
		gotText = body["text"].(string)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})

	bot.OnThinking(context.Background())
	if gotText != "🧠 _思考中\\.\\.\\._" {
		t.Errorf("OnThinking text = %q", gotText)
	}
}

func TestOnToolCall(t *testing.T) {
	var gotText string
	bot, _ := captureRequest(t, func(w http.ResponseWriter, r *http.Request) {
		body := parseSendMessageBody(t, r)
		gotText = body["text"].(string)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})

	bot.OnToolCall(context.Background(), "read_file", `{"path":"a.txt"}`)
	// 应包含工具名
	if !bytes.Contains([]byte(gotText), []byte("read\\_file")) {
		t.Errorf("OnToolCall text missing tool name: %q", gotText)
	}
}

func TestOnToolResult(t *testing.T) {
	var gotText string
	bot, _ := captureRequest(t, func(w http.ResponseWriter, r *http.Request) {
		body := parseSendMessageBody(t, r)
		gotText = body["text"].(string)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})

	// 成功结果
	bot.OnToolResult(context.Background(), "read_file", "file content here", false)
	if !bytes.Contains([]byte(gotText), []byte("✅")) {
		t.Errorf("OnToolResult success text missing checkmark: %q", gotText)
	}

	// 错误结果 — 重新捕获
	var gotText2 string
	bot2, _ := captureRequest(t, func(w http.ResponseWriter, r *http.Request) {
		body := parseSendMessageBody(t, r)
		gotText2 = body["text"].(string)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})

	bot2.OnToolResult(context.Background(), "bash", "command not found", true)
	if !bytes.Contains([]byte(gotText2), []byte("❌")) {
		t.Errorf("OnToolResult error text missing X: %q", gotText2)
	}
}

func TestOnMessage(t *testing.T) {
	var gotText string
	bot, _ := captureRequest(t, func(w http.ResponseWriter, r *http.Request) {
		body := parseSendMessageBody(t, r)
		gotText = body["text"].(string)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})

	bot.OnMessage(context.Background(), "任务完成！")
	if !bytes.Contains([]byte(gotText), []byte("任务完成")) {
		t.Errorf("OnMessage text missing content: %q", gotText)
	}
}

func TestSendFailureDoesNotPanic(t *testing.T) {
	// 服务器返回 500，不应 panic
	bot, _ := captureRequest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	// 只要不 panic 就行
	bot.OnMessage(context.Background(), "test")
}
```

注意：在 `telegram_bot_test.go` 顶部 import 中添加 `"bytes"` 和 `"context"`。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/reporter/ -run "TestOnThinking|TestOnToolCall|TestOnToolResult|TestOnMessage|TestSendFailure" -v`
Expected: 编译失败，Reporter 方法未实现

- [ ] **Step 3: 实现四个 Reporter 方法**

在 `telegram_bot.go` 末尾追加：

```go
import (
	"context"
	"fmt"
)

func (b *TelegramBot) OnThinking(ctx context.Context) {
	_ = ctx
	text := "🧠 _思考中\\.\\.\\_"
	if err := b.sendText(text); err != nil {
		log.Printf("[telegram] OnThinking 发送失败: %v", err)
	}
}

func (b *TelegramBot) OnToolCall(ctx context.Context, toolName string, args string) {
	_ = ctx
	escaped := escapeMarkdownV2(truncate(args, 200))
	text := fmt.Sprintf("⚙ *调用工具:* `%s`\n```\n%s\n```", escapeMarkdownV2(toolName), escaped)
	if err := b.sendText(text); err != nil {
		log.Printf("[telegram] OnToolCall 发送失败: %v", err)
	}
}

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

func (b *TelegramBot) OnMessage(ctx context.Context, content string) {
	_ = ctx
	text := "💬 " + escapeMarkdownV2(truncate(content, 4000))
	if err := b.sendText(text); err != nil {
		log.Printf("[telegram] OnMessage 发送失败: %v", err)
	}
}
```

注意：确保 `telegram_bot.go` 顶部 import 合并后包含 `"context"` 和 `"log"`。移除重复的 `"fmt"` 导入。

- [ ] **Step 4: 运行全部测试确认通过**

Run: `go test ./internal/reporter/ -v`
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/reporter/telegram_bot.go internal/reporter/telegram_bot_test.go
git commit -m "feat(reporter): 实现 TelegramBot Reporter 接口四个方法"
```

---

### Task 4: 集成到 main.go 与配置

**Files:**
- Modify: `internal/config/config.go`
- Modify: `cmd/claw/main.go`

- [ ] **Step 1: 添加 Telegram 配置结构**

在 `internal/config/config.go` 中，在 `Config` 结构体添加 Telegram 字段：

```go
type Config struct {
	LLM      LLM      `mapstructure:"llm"`
	Telegram Telegram `mapstructure:"telegram"`
}

type Telegram struct {
	Token  string `mapstructure:"token"`
	ChatID string `mapstructure:"chat_id"`
}
```

- [ ] **Step 2: 在 main.go 中集成 reporter**

在 `cmd/claw/main.go` 中：
- import 添加 `"github.com/yanking/go-tiny-claw/internal/reporter"`
- 在 engine 创建之后、`eng.Run` 之前，构造 TelegramBot：

```go
	var r reporter.Reporter
	if c.Telegram.Token != "" && c.Telegram.ChatID != "" {
		r = reporter.NewTelegramBot(c.Telegram.Token, c.Telegram.ChatID)
		log.Println("[main] Telegram reporter 已启用")
	}
```

- 修改 `eng.Run` 调用，传入 reporter：

```go
	err := eng.Run(context.Background(), prompt, r)
```

注意：当前 `main.go` 中 `eng.Run` 的签名可能还没有 reporter 参数，需确认与 `loop.go:32` 一致。根据代码 review，`loop.go:32` 已包含 reporter 参数。

- [ ] **Step 3: 运行测试确认无编译错误**

Run: `go build ./...`
Expected: 编译成功

- [ ] **Step 4: 提交**

```bash
git add internal/config/config.go cmd/claw/main.go
git commit -m "feat: 集成 Telegram reporter 到配置与入口"
```

---

## 自审检查清单

- [x] Spec 覆盖：Reporter 接口 4 个方法 ✅，MarkdownV2 转义 ✅，内容截断 ✅，错误处理不中断 ✅，配置 ✅
- [x] 无占位符：每个 step 都有完整代码
- [x] 类型一致：所有函数签名在定义和使用处保持一致
