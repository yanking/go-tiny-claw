# Telegram Bot Reporter 设计

## 概述

实现 `reporter.Reporter` 接口，将 Agent 的运行过程（思考、工具调用、工具结果、最终回复）推送到 Telegram 聊天。仅做单向推送，不处理用户输入。

## 依赖

- 标准库 `net/http` 调用 Telegram Bot API
- 零新增外部依赖

## 接口实现

```go
type TelegramBot struct {
    token  string
    chatID string
    client *http.Client
}

func NewTelegramBot(token, chatID string) *TelegramBot
```

四个方法均通过 `POST https://api.telegram.org/bot{token}/sendMessage` 发送消息，使用 `parse_mode=MarkdownV2`。

### 消息格式

| 方法 | 消息内容 |
|------|---------|
| `OnThinking` | `🧠 _思考中\.\.\._` |
| `OnToolCall` | `⚙ *调用工具:* \`{name}\`\n\`{args前200字符}\`` |
| `OnToolResult` | `✅ *{name} 结果:*\n\`\`\`\n{result前1000字符}\n\`\`\`` (error 时用 ❌) |
| `OnMessage` | `💬 {content}` |

### 内容截断

- 工具参数：最多 200 字符
- 工具结果：最多 1000 字符
- 最终消息：最多 4000 字符（Telegram 单条消息上限约 4096）

### MarkdownV2 转义

对用户内容中的 `_ * [ ] ( ) ~ \` > # + - = | { } . !` 字符进行转义，避免解析失败。

### 错误处理

发送失败时仅 `log.Printf` 记录错误，不影响 Agent 主流程。reporter 是旁路通知，不应中断核心逻辑。

## 配置

Bot token 和 chat ID 通过构造函数参数传入，配置来源由调用方（main.go）决定。后续可在 `config.Config` 中添加 telegram 配置段。

## 文件

- `internal/reporter/telegram_bot.go` — 唯一新增/修改文件
