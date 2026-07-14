package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hotuyen21pt/work_reminder/config"
	"github.com/hotuyen21pt/work_reminder/internal/telegram"
)

func main() {
	configPath := getenvDefault("CONFIG_PATH", "config/config.yml")

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	reminder, err := pickReminder(cfg, os.Getenv("REMINDER_NAME"))
	if err != nil {
		log.Fatalf("pick reminder: %v", err)
	}

	group, ok := cfg.Groups[reminder.Group]
	if !ok {
		log.Fatalf("reminder %q references unknown group %q", reminder.Name, reminder.Group)
	}

	threadID, err := parseThreadID(group.MessageThreadID)
	if err != nil {
		log.Fatalf("invalid message_thread_id for group %q: %v", reminder.Group, err)
	}

	message := renderMessage(reminder.Text)

	client := telegram.New(cfg.Token)
	if reminder.Photo != "" {
		if err := client.SendPhoto(group.ChatID, reminder.Photo, message, threadID); err != nil {
			log.Fatalf("send photo: %v", err)
		}
	} else {
		if err := client.SendMessage(group.ChatID, message, threadID); err != nil {
			log.Fatalf("send message: %v", err)
		}
	}

	log.Printf("sent reminder %q to group %q (chat_id=%s)", reminder.Name, reminder.Group, group.ChatID)
}

// renderMessage ghép tiêu đề ngày động (thứ + ngày/tháng/năm theo giờ VN) vào
// trước phần nội dung tĩnh lấy từ config.
func renderMessage(body string) string {
	now := nowVN()
	header := fmt.Sprintf("🗓 <b>%s · %s</b>", weekdayVN(now.Weekday()), now.Format("02/01/2006"))
	return header + "\n" + body
}

// nowVN trả về thời điểm hiện tại theo giờ Việt Nam. Nếu không nạp được tzdata
// (một số môi trường Windows) thì fallback về UTC+7 cố định.
func nowVN() time.Time {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.FixedZone("ICT", 7*60*60)
	}
	return time.Now().In(loc)
}

// weekdayVN đổi thứ trong tuần sang cách gọi tiếng Việt (Thứ 2 .. Chủ nhật).
func weekdayVN(d time.Weekday) string {
	switch d {
	case time.Monday:
		return "Thứ 2"
	case time.Tuesday:
		return "Thứ 3"
	case time.Wednesday:
		return "Thứ 4"
	case time.Thursday:
		return "Thứ 5"
	case time.Friday:
		return "Thứ 6"
	case time.Saturday:
		return "Thứ 7"
	default:
		return "Chủ nhật"
	}
}

// pickReminder chọn reminder theo tên. Nếu name rỗng thì lấy cái đầu tiên.
func pickReminder(cfg *config.Config, name string) (config.Reminder, error) {
	if len(cfg.Reminders) == 0 {
		return config.Reminder{}, fmt.Errorf("no reminders defined in config")
	}
	if name == "" {
		return cfg.Reminders[0], nil
	}
	for _, r := range cfg.Reminders {
		if r.Name == name {
			return r, nil
		}
	}
	return config.Reminder{}, fmt.Errorf("no reminder named %q", name)
}

// parseThreadID chuyển message_thread_id (chuỗi, có thể rỗng) sang int.
func parseThreadID(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
