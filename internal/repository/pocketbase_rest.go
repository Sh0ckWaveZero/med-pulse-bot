// Package repository provides PocketBase REST API implementations
package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"telegram-bot-med/internal/models"
)

// PocketBaseRESTEmployeeRepository implements EmployeeRepository
type PocketBaseRESTEmployeeRepository struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewPocketBaseRESTEmployeeRepository creates repository
func NewPocketBaseRESTEmployeeRepository(baseURL string) *PocketBaseRESTEmployeeRepository {
	return &PocketBaseRESTEmployeeRepository{
		baseURL:    strings.TrimRight(baseURL, "/"),
		authToken:  os.Getenv("POCKETBASE_TOKEN"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (r *PocketBaseRESTEmployeeRepository) addAuthHeader(req *http.Request) {
	if r.authToken != "" {
		req.Header.Set("Authorization", r.authToken)
	}
}

func (r *PocketBaseRESTEmployeeRepository) GetByMacAddress(ctx context.Context, macAddress string) (*models.Employee, error) {
	filter := fmt.Sprintf("mac_address='%s' && is_active=true", strings.ToLower(macAddress))
	encodedFilter := url.QueryEscape(filter)
	apiURL := fmt.Sprintf("%s/api/collections/employees/records?filter=%s&limit=1", r.baseURL, encodedFilter)

	log.Printf("🔍 Looking up employee by MAC: %s", macAddress)
	log.Printf("🔍 API URL: %s", apiURL)

	req, _ := http.NewRequest("GET", apiURL, nil)
	r.addAuthHeader(req)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		log.Printf("❌ HTTP error looking up employee: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("🔍 API Response Status: %d", resp.StatusCode)
	log.Printf("🔍 API Response Body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get employee: %s", resp.Status)
	}

	// Re-create reader for JSON decoding
	resp.Body = io.NopCloser(strings.NewReader(string(body)))

	var result struct {
		Items []struct {
			ID             string `json:"id"`
			MacAddress     string `json:"mac_address"`
			TelegramChatID int64  `json:"telegram_chat_id"`
			Name           string `json:"name"`
			WorkStartTime  string `json:"work_start_time"`
			IsActive       bool   `json:"is_active"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("employee not found")
	}

	item := result.Items[0]
	return &models.Employee{
		ID:             item.ID,
		TelegramChatID: item.TelegramChatID,
		Name:           item.Name,
		MacAddress:     item.MacAddress,
		WorkStartTime:  item.WorkStartTime,
		IsActive:       item.IsActive,
	}, nil
}

func (r *PocketBaseRESTEmployeeRepository) IsCheckedInToday(ctx context.Context, employeeID string) (bool, error) {
	today := time.Now().Format("2006-01-02")
	filter := fmt.Sprintf("employee_id='%s' && created_date='%s'", employeeID, today)
	encodedFilter := url.QueryEscape(filter)
	apiURL := fmt.Sprintf("%s/api/collections/attendance/records?filter=%s&limit=1", r.baseURL, encodedFilter)

	log.Printf("🔍 Checking attendance for employee ID %s on %s", employeeID, today)
	log.Printf("🔍 Attendance API URL: %s", apiURL)

	req, _ := http.NewRequest("GET", apiURL, nil)
	r.addAuthHeader(req)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		log.Printf("❌ HTTP error checking attendance: %v", err)
		return false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("🔍 Attendance Response Status: %d", resp.StatusCode)
	log.Printf("🔍 Attendance Response Body: %s", string(body))

	var result struct {
		Items []interface{} `json:"items"`
	}

	if err := json.NewDecoder(strings.NewReader(string(body))).Decode(&result); err != nil {
		log.Printf("❌ JSON decode error: %v", err)
		return false, err
	}

	isCheckedIn := len(result.Items) > 0
	log.Printf("✅ Employee ID %s checked in today: %v", employeeID, isCheckedIn)
	return isCheckedIn, nil
}

// PocketBaseRESTAttendanceRepository implements AttendanceRepository
type PocketBaseRESTAttendanceRepository struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

func NewPocketBaseRESTAttendanceRepository(baseURL string) *PocketBaseRESTAttendanceRepository {
	return &PocketBaseRESTAttendanceRepository{
		baseURL:    strings.TrimRight(baseURL, "/"),
		authToken:  os.Getenv("POCKETBASE_TOKEN"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (r *PocketBaseRESTAttendanceRepository) addAuthHeader(req *http.Request) {
	if r.authToken != "" {
		req.Header.Set("Authorization", r.authToken)
	}
}

func (r *PocketBaseRESTAttendanceRepository) Create(ctx context.Context, attendance *models.Attendance) error {
	url := fmt.Sprintf("%s/api/collections/attendance/records", r.baseURL)

	data := map[string]interface{}{
		"employee_id":   attendance.EmployeeID,
		"check_in_time": attendance.CheckInTime.Format(time.RFC3339),
		"scanner_mac":   attendance.ScannerMac,
		"status":        attendance.Status,
		"created_date":  attendance.CreatedDate.Format("2006-01-02"),
	}

	jsonData, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	r.addAuthHeader(req)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create attendance: %s - %s", resp.Status, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	attendance.ID = result.ID
	return nil
}

// PocketBaseRESTDetectionRepository implements EmployeeDetectionRepository
type PocketBaseRESTDetectionRepository struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

func NewPocketBaseRESTDetectionRepository(baseURL string) *PocketBaseRESTDetectionRepository {
	return &PocketBaseRESTDetectionRepository{
		baseURL:    strings.TrimRight(baseURL, "/"),
		authToken:  os.Getenv("POCKETBASE_TOKEN"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (r *PocketBaseRESTDetectionRepository) addAuthHeader(req *http.Request) {
	if r.authToken != "" {
		req.Header.Set("Authorization", r.authToken)
	}
}

func (r *PocketBaseRESTDetectionRepository) Create(ctx context.Context, detection *models.EmployeeDetection) error {
	url := fmt.Sprintf("%s/api/collections/employee_detections/records", r.baseURL)

	data := map[string]interface{}{
		"employee_id":      detection.EmployeeID,
		"mac_address":      strings.ToLower(detection.MacAddress),
		"scanner_mac":      detection.ScannerMac,
		"rssi":             detection.RSSI,
		"device_type":      detection.DeviceType,
		"is_itag03":        detection.IsITag03,
		"is_target_device": detection.IsTargetDevice,
		"device_name":      detection.DeviceName,
		"detected_at":      detection.DetectedAt.Format(time.RFC3339),
	}

	jsonData, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	r.addAuthHeader(req)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create detection: %s - %s", resp.Status, string(body))
	}

	log.Printf("💾 Saved detection for employee ID %s: MAC=%s, RSSI=%d, Type=%s",
		detection.EmployeeID, detection.MacAddress, detection.RSSI, detection.DeviceType)

	return nil
}

// PocketBaseRESTScannerRepository implements ScannerRepository
type PocketBaseRESTScannerRepository struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

func NewPocketBaseRESTScannerRepository(baseURL string) *PocketBaseRESTScannerRepository {
	return &PocketBaseRESTScannerRepository{
		baseURL:    strings.TrimRight(baseURL, "/"),
		authToken:  os.Getenv("POCKETBASE_TOKEN"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (r *PocketBaseRESTScannerRepository) addAuthHeader(req *http.Request) {
	if r.authToken != "" {
		req.Header.Set("Authorization", r.authToken)
	}
}

func (r *PocketBaseRESTScannerRepository) UpdateActivity(ctx context.Context, scannerMac string) error {
	filter := fmt.Sprintf("scanner_mac='%s'", scannerMac)
	findURL := fmt.Sprintf("%s/api/collections/scanners/records?filter=%s&limit=1", r.baseURL, filter)

	req, _ := http.NewRequest("GET", findURL, nil)
	r.addAuthHeader(req)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
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
		updateURL := fmt.Sprintf("%s/api/collections/scanners/records/%s", r.baseURL, findResult.Items[0].ID)
		req, _ := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		r.addAuthHeader(req)
		resp, err = r.httpClient.Do(req)
	} else {
		createURL := fmt.Sprintf("%s/api/collections/scanners/records", r.baseURL)
		req, _ := http.NewRequest("POST", createURL, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		r.addAuthHeader(req)
		resp, err = r.httpClient.Do(req)
	}

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update scanner: %s - %s", resp.Status, string(body))
	}

	return nil
}
