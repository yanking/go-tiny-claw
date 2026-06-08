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
