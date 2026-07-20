package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Group là đích gửi (một group/chat trên Telegram).
type Group struct {
	ChatID          string `yaml:"chat_id"`
	MessageThreadID string `yaml:"message_thread_id"`
}

// Weather cấu hình lấy thời tiết hôm nay qua Open-Meteo (API public, không cần key).
type Weather struct {
	Lat   float64 `yaml:"lat"`
	Lon   float64 `yaml:"lon"`
	Label string  `yaml:"label"` // tên hiển thị, vd "Ho Chi Minh City"
}

// Reminder là một nội dung nhắc nhở, gắn với một group.
type Reminder struct {
	Name  string `yaml:"name"`
	Group string `yaml:"group"`
	Text  string `yaml:"text"`
	// Photo bật/tắt gửi kèm ảnh. true thì gửi dạng sendPhoto với một ảnh ngẫu nhiên
	// (sinh runtime), caption là Text.
	Photo bool `yaml:"photo"`
}

// Config là toàn bộ cấu hình đọc từ config.yml (+ token từ env).
type Config struct {
	Token string `yaml:"-"`
	// Dice là danh sách emoji động (🎲 🎯 🏀 ⚽ 🎳 🎰). Mỗi lần gửi chọn ngẫu nhiên 1 cái.
	// Danh sách rỗng = tắt. Chỉ 6 emoji này được Telegram sendDice hỗ trợ.
	Dice []string `yaml:"dice"`
	// Reaction là emoji bot tự thả lên chính tin vừa gửi (vd 🔥). Để trống = tắt.
	Reaction string `yaml:"reaction"`
	// PlanStart là ngày bắt đầu kế hoạch (YYYY-MM-DD) để tính "Ngày thứ N". Để trống = tắt.
	PlanStart string `yaml:"plan_start"`
	// QuoteAPI là URL API trả JSON zenquotes ([{"q","a"}]). Để trống = dùng quote fallback.
	QuoteAPI string `yaml:"quote_api"`
	// Weather bật hiển thị thời tiết hôm nay (Open-Meteo). Bỏ khối này = tắt.
	Weather   *Weather         `yaml:"weather"`
	Groups    map[string]Group `yaml:"groups"`
	Reminders []Reminder       `yaml:"reminders"`
}

// Load đọc file yaml và lấy token từ biến môi trường BOT_TOKEN.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("missing BOT_TOKEN environment variable")
	}
	cfg.Token = token

	return &cfg, nil
}
