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

// sendDiceRequest là payload gửi lên endpoint sendDice (emoji động: 🎲 🎯 🏀 ⚽ 🎳 🎰).
type sendDiceRequest struct {
	ChatID          string `json:"chat_id"`
	Emoji           string `json:"emoji,omitempty"`
	MessageThreadID int    `json:"message_thread_id,omitempty"`
}

// reactionType là một phần tử trong mảng reaction của setMessageReaction.
type reactionType struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}

// setMessageReactionRequest là payload gửi lên endpoint setMessageReaction.
type setMessageReactionRequest struct {
	ChatID    string         `json:"chat_id"`
	MessageID int            `json:"message_id"`
	Reaction  []reactionType `json:"reaction"`
}

// SendMessage gửi một tin nhắn HTML vào chat/thread chỉ định, trả về message_id.
// threadID = 0 nghĩa là gửi vào kênh chính (không dùng topic).
func (c *Client) SendMessage(chatID, text string, threadID int) (int, error) {
	return c.call("sendMessage", sendMessageRequest{
		ChatID:          chatID,
		Text:            text,
		ParseMode:       "HTML",
		MessageThreadID: threadID,
	})
}

// SendPhoto gửi một ảnh (photo là URL) kèm caption HTML, trả về message_id.
func (c *Client) SendPhoto(chatID, photoURL, caption string, threadID int) (int, error) {
	return c.call("sendPhoto", sendPhotoRequest{
		ChatID:          chatID,
		Photo:           photoURL,
		Caption:         caption,
		ParseMode:       "HTML",
		MessageThreadID: threadID,
	})
}

// SendDice gửi một emoji động (🎲 🎯 🏀 ⚽ 🎳 🎰) — Telegram chơi animation rồi dừng ở
// một giá trị ngẫu nhiên. emoji rỗng thì mặc định là 🎲.
func (c *Client) SendDice(chatID, emoji string, threadID int) (int, error) {
	return c.call("sendDice", sendDiceRequest{
		ChatID:          chatID,
		Emoji:           emoji,
		MessageThreadID: threadID,
	})
}

// SetMessageReaction thả một emoji reaction lên một message đã gửi.
func (c *Client) SetMessageReaction(chatID string, messageID int, emoji string) error {
	_, err := c.call("setMessageReaction", setMessageReactionRequest{
		ChatID:    chatID,
		MessageID: messageID,
		Reaction:  []reactionType{{Type: "emoji", Emoji: emoji}},
	})
	return err
}

// call gửi payload JSON tới một method của Bot API, kiểm tra status và trả về
// message_id trong result (0 nếu response không có, ví dụ setMessageReaction).
func (c *Client) call(method string, payload any) (int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", c.token, method)
	resp, err := c.http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("call telegram api: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("telegram api %s returned %d: %s", method, resp.StatusCode, string(data))
	}

	var parsed struct {
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	_ = json.Unmarshal(data, &parsed) // result có thể là bool (vd setMessageReaction) -> bỏ qua lỗi
	return parsed.Result.MessageID, nil
}
