package reporter

import (
	"bytes"
	"context"
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

// captureRequest 启动一个 mock Telegram API 服务器，返回 bot 和 server 实例
func captureRequest(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) (*TelegramBot, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(ts.Close)
	bot := NewTelegramBot("123:ABC", "999")
	bot.apiBase = ts.URL
	bot.client = ts.Client()
	return bot, ts
}

// parseSendMessageBody 解码 sendMessage 请求的 JSON body
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
		{`a\b`, `a\\b`},
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
	longBytes := make([]byte, 100)
	for i := range longBytes {
		longBytes[i] = 'x'
	}
	long := string(longBytes)
	got := truncate(long, 50)
	if len(got) > 65 { // 50 + "\n...(truncated)" 长度
		t.Errorf("truncate long len = %d, too long", len(got))
	}
	if len(got) < 50 {
		t.Errorf("truncate long len = %d, should keep first 50", len(got))
	}
}

func TestOnThinking(t *testing.T) {
	var gotText string
	bot, _ := captureRequest(t, func(w http.ResponseWriter, r *http.Request) {
		body := parseSendMessageBody(t, r)
		gotText = body["text"].(string)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})

	bot.OnThinking(context.Background())
	if gotText != "🧠 思考中\\.\\.\\." {
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
	// 应包含转义后的工具名
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

	// 错误结果
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
	bot, _ := captureRequest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	// 只要不 panic 就行
	bot.OnMessage(context.Background(), "test")
}
