package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client là wrapper mỏng quanh Telegram Bot API.
type Client struct {
	token string
	http  *http.Client
}

// New tạo client với bot token.
func New(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 15 * time.Second},
	}
}

// sendMessageRequest là payload gửi lên endpoint sendMessage.
type sendMessageRequest struct {
	ChatID          string `json:"chat_id"`
	Text            string `json:"text"`
	ParseMode       string `json:"parse_mode"`
	MessageThreadID int    `json:"message_thread_id,omitempty"`
}

// sendPhotoRequest là payload gửi lên endpoint sendPhoto.
type sendPhotoRequest struct {
	ChatID          string `json:"chat_id"`
	Photo           string `json:"photo"`
	Caption         string `json:"caption,omitempty"`
	ParseMode       string `json:"parse_mode"`
	MessageThreadID int    `json:"message_thread_id,omitempty"`
}

// SendMessage gửi một tin nhắn HTML vào chat/thread chỉ định.
// threadID = 0 nghĩa là gửi vào kênh chính (không dùng topic).
func (c *Client) SendMessage(chatID, text string, threadID int) error {
	return c.call("sendMessage", sendMessageRequest{
		ChatID:          chatID,
		Text:            text,
		ParseMode:       "HTML",
		MessageThreadID: threadID,
	})
}

// SendPhoto gửi một ảnh (photo là URL) kèm caption HTML vào chat/thread chỉ định.
func (c *Client) SendPhoto(chatID, photoURL, caption string, threadID int) error {
	return c.call("sendPhoto", sendPhotoRequest{
		ChatID:          chatID,
		Photo:           photoURL,
		Caption:         caption,
		ParseMode:       "HTML",
		MessageThreadID: threadID,
	})
}

// call gửi payload JSON tới một method của Bot API và kiểm tra HTTP status.
func (c *Client) call(method string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", c.token, method)
	resp, err := c.http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("call telegram api: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api %s returned %d: %s", method, resp.StatusCode, string(data))
	}
	return nil
}
