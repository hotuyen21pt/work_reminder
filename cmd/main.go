package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
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

	message := renderMessage(reminder.Text, planDay(cfg.PlanStart, nowVN()), fetchWeather(cfg.Weather), fetchQuote(cfg.QuoteAPI))

	client := telegram.New(cfg.Token)
	messageID := sendReminder(client, reminder, group, message, threadID)

	// Bot tự thả reaction lên chính tin vừa gửi. Lỗi/không có message_id -> bỏ qua.
	if cfg.Reaction != "" && messageID != 0 {
		if err := client.SetMessageReaction(group.ChatID, messageID, cfg.Reaction); err != nil {
			log.Printf("set reaction failed (bỏ qua): %v", err)
		}
	}

	// Dice là emoji động gửi kèm cho vui, chọn ngẫu nhiên từ danh sách để đa dạng.
	// Lỗi không được làm hỏng reminder chính.
	if len(cfg.Dice) > 0 {
		emoji := cfg.Dice[int(time.Now().UnixNano())%len(cfg.Dice)]
		if _, err := client.SendDice(group.ChatID, emoji, threadID); err != nil {
			log.Printf("send dice failed (bỏ qua): %v", err)
		}
	}

	log.Printf("sent reminder %q to group %q (chat_id=%s)", reminder.Name, reminder.Group, group.ChatID)
}

// sendReminder gửi tin chính dưới dạng ảnh tĩnh (sendPhoto) kèm caption. Nếu gửi ảnh
// lỗi thì fallback text-only để reminder không bao giờ bị mất.
func sendReminder(client *telegram.Client, reminder config.Reminder, group config.Group, message string, threadID int) int {
	if !reminder.Photo {
		id, err := client.SendMessage(group.ChatID, message, threadID)
		if err != nil {
			log.Fatalf("send message: %v", err)
		}
		return id
	}

	if id, err := client.SendPhoto(group.ChatID, randomPhotoURL(), message, threadID); err == nil {
		return id
	} else {
		log.Printf("send photo failed, fallback to text: %v", err)
	}

	id, err := client.SendMessage(group.ChatID, message, threadID)
	if err != nil {
		log.Fatalf("send message (final fallback): %v", err)
	}
	return id
}

// randomPhotoURL sinh URL ảnh ngẫu nhiên từ picsum.photos. Tham số random đổi
// theo thời điểm chạy (nano) nên mỗi lần gửi Telegram tải về một ảnh khác nhau.
func randomPhotoURL() string {
	return fmt.Sprintf("https://picsum.photos/800/500?random=%d", time.Now().UnixNano())
}

// renderMessage ghép tiêu đề động (thứ + ngày + "Day N" của kế hoạch) trước phần
// nội dung từ config, và gắn quote-of-the-day ở cuối. Nội dung hiển thị tiếng Anh.
func renderMessage(body string, dayNum int, weather, quote string) string {
	now := nowVN()
	header := fmt.Sprintf("🗓 <b>%s · %s</b>", now.Weekday().String(), now.Format("02 Jan 2006"))
	if dayNum > 0 {
		header += fmt.Sprintf("\n🔥 <b>Day %d</b> of the plan", dayNum)
	}
	if weather != "" {
		header += "\n" + weather
	}
	// divider là một dòng dài cố định để "kéo rộng" bong bóng tin (Telegram lấy độ
	// rộng theo dòng dài nhất). Đặt giữa header và nội dung, và trước quote.
	const divider = "━━━━━━━━━━━━━━━━━━━━━━━"
	msg := header + "\n" + divider + "\n" + body
	if quote != "" {
		msg += "\n" + divider + "\n💬 <i>" + quote + "</i>"
	}
	return msg
}

// planDay tính "hôm nay là ngày thứ mấy" kể từ ngày bắt đầu kế hoạch (đã tính theo
// giờ VN). Ngày bắt đầu = Ngày 1. Trả 0 nếu không cấu hình / ngày lỗi / chưa tới.
func planDay(start string, now time.Time) int {
	if start == "" {
		return 0
	}
	s, err := time.ParseInLocation("2006-01-02", start, now.Location())
	if err != nil {
		return 0
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(today.Sub(s).Hours()/24) + 1
	if days < 1 {
		return 0
	}
	return days
}

// fetchQuote lấy một câu quote ngẫu nhiên từ API (định dạng zenquotes: [{"q","a"}]).
// Nếu không có API / lỗi mạng / phản hồi hỏng thì rơi về danh sách quote tiếng Việt
// dựng sẵn — để reminder luôn có quote. Kết quả đã được escape cho HTML parse mode.
func fetchQuote(apiURL string) string {
	fallback := []string{
		"Success is the sum of small efforts repeated day in and day out.",
		"Discipline is the bridge between goals and accomplishment.",
		"Don't count the days, make the days count.",
		"It's okay to go slow, just don't stop.",
		"Today is hard, tomorrow is harder, but the day after is wonderful.",
		"What you do today decides who you become tomorrow.",

		"Small progress is still progress.",
		"Consistency beats intensity.",
		"Dream big. Start small. Act now.",
		"One hour today is better than zero hours.",
		"Your future self will thank you.",
		"The expert was once a beginner.",
		"Focus on progress, not perfection.",
		"Every study session is an investment in yourself.",
		"Don't wish for it. Work for it.",
		"Be better than you were yesterday.",
		"Success starts with self-discipline.",
		"The pain of discipline is lighter than the pain of regret.",
		"Keep showing up, even on difficult days.",
		"Hard work compounds like interest.",
		"The best time to start was yesterday. The next best time is now.",
		"Stay patient. Great things take time.",
		"Do something today that your future self will be proud of.",
		"Your only competition is who you were yesterday.",
		"Consistency creates confidence.",
		"Tiny improvements every day lead to remarkable results.",
		"Progress, not excuses.",
		"The difference between ordinary and extraordinary is that little extra.",
		"You don't have to be perfect, just keep improving.",
		"Every master was once a beginner.",
		"Keep learning. Keep growing.",
		"Stay focused and never give up.",
		"The comeback is always stronger than the setback.",
		"Your habits shape your future.",
		"Great achievements are built one day at a time.",
		"Success comes to those who refuse to quit.",
		"Believe in the process.",
		"Learning never exhausts the mind.",
		"Stay hungry. Stay curious.",
		"Make today so productive that tomorrow becomes easier.",
		"The more you practice, the luckier you become.",
		"Little by little, a little becomes a lot.",
		"Discipline today. Freedom tomorrow.",
		"Every English sentence you speak makes you more confident.",
		"Every line of code makes you a better engineer.",
		"Practice English. Build projects. Repeat.",
		"Fluency is built one conversation at a time.",
		"Code, learn, improve, repeat.",
		"A better career starts with today's study session.",
		"One solved problem is one step closer to mastery.",
		"Read more. Code more. Speak more.",
	}
	pick := fallback[int(time.Now().UnixNano())%len(fallback)]
	if apiURL == "" {
		return pick
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return pick
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return pick
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return pick
	}

	var zq []struct {
		Q string `json:"q"`
		A string `json:"a"`
	}
	if err := json.Unmarshal(data, &zq); err != nil || len(zq) == 0 || zq[0].Q == "" {
		return pick
	}
	q := zq[0].Q
	if zq[0].A != "" {
		q += " — " + zq[0].A
	}
	if len([]rune(q)) > 200 {
		q = string([]rune(q)[:197]) + "…"
	}
	return html.EscapeString(q)
}

// fetchWeather lấy thời tiết hôm nay qua Open-Meteo (không cần API key). Trả về một
// dòng đã format (emoji + nhiệt độ + mô tả + min/max), hoặc "" nếu tắt/lỗi.
func fetchWeather(w *config.Weather) string {
	if w == nil {
		return ""
	}
	endpoint := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%g&longitude=%g"+
		"&current=temperature_2m,weather_code&daily=temperature_2m_max,temperature_2m_min"+
		"&timezone=auto&forecast_days=1", w.Lat, w.Lon)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return ""
	}

	var r struct {
		Current struct {
			Temp float64 `json:"temperature_2m"`
			Code int     `json:"weather_code"`
		} `json:"current"`
		Daily struct {
			Max []float64 `json:"temperature_2m_max"`
			Min []float64 `json:"temperature_2m_min"`
		} `json:"daily"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return ""
	}

	label := w.Label
	if label == "" {
		label = "Today"
	}
	emoji, desc := weatherCode(r.Current.Code)
	line := fmt.Sprintf("%s <b>%s</b> · %.0f°C, %s", emoji, html.EscapeString(label), r.Current.Temp, desc)
	if len(r.Daily.Min) > 0 && len(r.Daily.Max) > 0 {
		line += fmt.Sprintf(" (%.0f°–%.0f°)", r.Daily.Min[0], r.Daily.Max[0])
	}
	return line
}

// weatherCode đổi mã WMO weather_code (Open-Meteo) sang emoji + mô tả tiếng Anh.
func weatherCode(code int) (string, string) {
	switch code {
	case 0:
		return "☀️", "Clear sky"
	case 1:
		return "🌤️", "Mainly clear"
	case 2:
		return "⛅", "Partly cloudy"
	case 3:
		return "☁️", "Overcast"
	case 45, 48:
		return "🌫️", "Fog"
	case 51, 53, 55:
		return "🌦️", "Drizzle"
	case 56, 57:
		return "🌧️", "Freezing drizzle"
	case 61, 63, 65:
		return "🌧️", "Rain"
	case 66, 67:
		return "🌧️", "Freezing rain"
	case 71, 73, 75, 77:
		return "🌨️", "Snow"
	case 80, 81, 82:
		return "🌦️", "Rain showers"
	case 85, 86:
		return "🌨️", "Snow showers"
	case 95:
		return "⛈️", "Thunderstorm"
	case 96, 99:
		return "⛈️", "Thunderstorm with hail"
	default:
		return "🌡️", "Weather"
	}
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
