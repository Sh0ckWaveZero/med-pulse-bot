package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	bot          *tgbotapi.BotAPI
	targetChatID int64
	pbURL        string
	pbToken      string
	httpClient   = &http.Client{Timeout: 10 * time.Second}
	userStates   = make(map[int64]*RegistrationState)
)

type RegistrationState struct {
	Step         int
	MacAddress   string
	Name         string
	EmployeeCode string
	Department   string
}

// SetPocketBaseURL sets the PocketBase REST API URL
func SetPocketBaseURL(url string) {
	pbURL = strings.TrimRight(url, "/")
}

// SetPocketBaseToken sets the PocketBase auth token
func SetPocketBaseToken(token string) {
	pbToken = token
}

// addAuthHeader adds authorization header if token exists
func addAuthHeader(req *http.Request) {
	if pbToken != "" {
		req.Header.Set("Authorization", pbToken)
	}
}

// Init initializes the Telegram Bot
func Init(token string, authorizedChatIDStr string) error {
	var err error
	bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	if authorizedChatIDStr != "" {
		id, err := strconv.ParseInt(authorizedChatIDStr, 10, 64)
		if err == nil {
			targetChatID = id
		}
	}

	return nil
}

// StartPolling starts the update loop
func StartPolling() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	go func() {
		for update := range updates {
			if update.CallbackQuery != nil {
				handleCallback(update.CallbackQuery)
				continue
			}

			if update.Message == nil || !update.Message.IsCommand() {
				continue
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			msg.ParseMode = "Markdown"

			switch update.Message.Command() {
			case "start":
				msg.Text = "🏢 *ระบบบันทึกเวลาเข้างาน*\n\n" +
					"*คำสั่ง:*\n" +
					"/register_employee - ลงทะเบียน\n" +
					"/myinfo - ข้อมูลฉัน\n" +
					"/today - เวลาวันนี้\n" +
					"/history - ประวัติ\n" +
					"/scanners - สถานะ Scanner"

			case "getid":
				msg.Text = fmt.Sprintf("Chat ID: `%d`", update.Message.Chat.ID)

			case "scanners":
				scanners, err := getActiveScanners()
				if err != nil {
					msg.Text = fmt.Sprintf("Error: %v", err)
				} else if len(scanners) == 0 {
					msg.Text = "No scanners found"
				} else {
					msg.Text = "📡 *Scanners:*\n" + strings.Join(scanners, "\n")
				}

			case "register_employee":
				handleRegisterEmployee(update.Message, &msg)

			case "myinfo":
				handleMyInfo(update.Message.Chat.ID, &msg)

			case "today":
				handleToday(update.Message.Chat.ID, &msg)

			case "history":
				handleHistory(update.Message, &msg)

			default:
				msg.Text = "ไม่รู้จำคำสั่ง ใช้ /start"
			}

			if _, err := bot.Send(msg); err != nil {
				log.Printf("Bot send error: %v", err)
			}
		}
	}()
}

func handleCallback(query *tgbotapi.CallbackQuery) {
	// Simplified callback handler
	callback := tgbotapi.NewCallback(query.ID, "OK")
	bot.Request(callback)
}

func handleRegisterEmployee(message *tgbotapi.Message, msg *tgbotapi.MessageConfig) {
	args := strings.Fields(message.CommandArguments())
	if len(args) < 4 {
		msg.Text = "Usage: `/register_employee <MAC> <Name> <Code> <Dept>`"
		return
	}

	err := registerEmployee(args[0], message.Chat.ID, args[1], args[2], strings.Join(args[3:], " "))
	if err != nil {
		msg.Text = fmt.Sprintf("❌ Error: %v", err)
	} else {
		msg.Text = fmt.Sprintf("✅ Registered!\nName: %s\nCode: %s", args[1], args[2])
	}
}

func handleMyInfo(chatID int64, msg *tgbotapi.MessageConfig) {
	emp, err := getEmployeeByChat(chatID)
	if err != nil {
		msg.Text = "❌ Not registered. Use /register_employee"
		return
	}
	msg.Text = fmt.Sprintf("👤 *Info*\nName: %s\nCode: %s\nDept: %s\nMAC: %s",
		emp.Name, emp.EmployeeCode, emp.Department, emp.MacAddress)
}

func handleToday(chatID int64, msg *tgbotapi.MessageConfig) {
	att, err := getTodayAttendance(chatID)
	if err != nil || att == nil {
		msg.Text = "No check-in today"
		return
	}
	msg.Text = fmt.Sprintf("📊 *Today*\nIn: %s\nStatus: %s",
		att.CheckInTime.Format("15:04"), att.Status)
}

func handleHistory(message *tgbotapi.Message, msg *tgbotapi.MessageConfig) {
	history, err := getAttendanceHistory(message.Chat.ID, 7)
	if err != nil || len(history) == 0 {
		msg.Text = "No history found"
		return
	}
	text := "📅 *History*\n\n"
	for _, h := range history {
		text += fmt.Sprintf("%s: %s\n", h.CreatedDate.Format("02/01"), h.Status)
	}
	msg.Text = text
}

// REST API Functions

func getActiveScanners() ([]string, error) {
	if pbURL == "" {
		return nil, fmt.Errorf("PocketBase URL not set")
	}

	url := fmt.Sprintf("%s/api/collections/scanners/records?sort=-last_seen", pbURL)
	req, _ := http.NewRequest("GET", url, nil)
	addAuthHeader(req)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ScannerMac string `json:"scanner_mac"`
			LastSeen   string `json:"last_seen"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var scanners []string
	for _, item := range result.Items {
		scanners = append(scanners, fmt.Sprintf("- `%s` (%s)", item.ScannerMac, item.LastSeen))
	}
	return scanners, nil
}

func registerEmployee(mac string, chatID int64, name, code, dept string) error {
	if pbURL == "" {
		return fmt.Errorf("PocketBase URL not set")
	}

	url := fmt.Sprintf("%s/api/collections/employees/records", pbURL)
	data := map[string]interface{}{
		"mac_address":      strings.ToUpper(mac),
		"telegram_chat_id": chatID,
		"name":             name,
		"employee_code":    code,
		"department":       dept,
		"is_active":        true,
	}

	jsonData, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(req)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, string(body))
	}
	return nil
}

func getEmployeeByChat(chatID int64) (*Employee, error) {
	if pbURL == "" {
		return nil, fmt.Errorf("PocketBase URL not set")
	}

	filter := fmt.Sprintf("telegram_chat_id=%d&&is_active=true", chatID)
	url := fmt.Sprintf("%s/api/collections/employees/records?filter=%s&limit=1", pbURL, filter)

	req, _ := http.NewRequest("GET", url, nil)
	addAuthHeader(req)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []Employee `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("not found")
	}

	return &result.Items[0], nil
}

func getTodayAttendance(chatID int64) (*Attendance, error) {
	emp, err := getEmployeeByChat(chatID)
	if err != nil {
		return nil, err
	}

	today := time.Now().Format("2006-01-02")
	filter := fmt.Sprintf("employee_id=%s&&created_date='%s'", emp.ID, today)
	url := fmt.Sprintf("%s/api/collections/attendance/records?filter=%s&sort=-check_in_time&limit=1", pbURL, filter)

	req, _ := http.NewRequest("GET", url, nil)
	addAuthHeader(req)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []Attendance `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	return &result.Items[0], nil
}

func getAttendanceHistory(chatID int64, days int) ([]Attendance, error) {
	emp, err := getEmployeeByChat(chatID)
	if err != nil {
		return nil, err
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	filter := fmt.Sprintf("employee_id=%s&&created_date>='%s'", emp.ID, startDate)
	url := fmt.Sprintf("%s/api/collections/attendance/records?filter=%s&sort=-created_date", pbURL, filter)

	req, _ := http.NewRequest("GET", url, nil)
	addAuthHeader(req)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []Attendance `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

// UpdateScannerActivity updates scanner via REST API
func UpdateScannerActivity(scannerMac string) {
	if pbURL == "" {
		return
	}

	// Try to find existing
	filter := fmt.Sprintf("scanner_mac='%s'", scannerMac)
	findURL := fmt.Sprintf("%s/api/collections/scanners/records?filter=%s&limit=1", pbURL, filter)

	req, _ := http.NewRequest("GET", findURL, nil)
	addAuthHeader(req)
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}

	var findResult struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&findResult)
	resp.Body.Close()

	data := map[string]interface{}{
		"scanner_mac": scannerMac,
		"last_seen":   time.Now().Format(time.RFC3339),
	}
	jsonData, _ := json.Marshal(data)

	if len(findResult.Items) > 0 {
		// Update
		updateURL := fmt.Sprintf("%s/api/collections/scanners/records/%s", pbURL, findResult.Items[0].ID)
		req, _ := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		addAuthHeader(req)
		httpClient.Do(req)
	} else {
		// Create
		createURL := fmt.Sprintf("%s/api/collections/scanners/records", pbURL)
		req, _ := http.NewRequest("POST", createURL, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		addAuthHeader(req)
		httpClient.Do(req)
	}
}

// SendNotification sends message to admin
func SendNotification(message string) {
	if bot == nil || targetChatID == 0 {
		return
	}
	msg := tgbotapi.NewMessage(targetChatID, message)
	msg.ParseMode = "Markdown"
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send: %v", err)
	}
}

// SendPersonalNotification sends to specific user
func SendPersonalNotification(chatID int64, message string) {
	if bot == nil {
		return
	}
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send to %d: %v", chatID, err)
	}
}

// Types
type Employee struct {
	ID             string `json:"id"`
	MacAddress     string `json:"mac_address"`
	TelegramChatID int64  `json:"telegram_chat_id"`
	Name           string `json:"name"`
	EmployeeCode   string `json:"employee_code"`
	Department     string `json:"department"`
	WorkStartTime  string `json:"work_start_time"`
	IsActive       bool   `json:"is_active"`
}

type Attendance struct {
	ID          string    `json:"id"`
	EmployeeID  string    `json:"employee_id"`
	CheckInTime time.Time `json:"check_in_time"`
	ScannerMac  string    `json:"scanner_mac"`
	Status      string    `json:"status"`
	CreatedDate time.Time `json:"created_date"`
}
