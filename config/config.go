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

// Reminder là một nội dung nhắc nhở, gắn với một group.
type Reminder struct {
	Name  string `yaml:"name"`
	Group string `yaml:"group"`
	Text  string `yaml:"text"`
	// Photo là URL ảnh (tùy chọn). Có ảnh thì gửi dạng sendPhoto với caption là Text.
	Photo string `yaml:"photo"`
}

// Config là toàn bộ cấu hình đọc từ config.yml (+ token từ env).
type Config struct {
	Token     string           `yaml:"-"`
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
