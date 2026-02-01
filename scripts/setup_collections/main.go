package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const pocketbaseURL = "http://192.168.100.100:8090"

var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	fmt.Println("🚀 PocketBase Collection Setup Script")
	fmt.Println("=====================================")

	// Load .env file if exists
	godotenv.Load()

	// Check if custom credentials provided via env
	url := getEnv("POCKETBASE_URL", pocketbaseURL)
	token := getEnv("POCKETBASE_TOKEN", "")

	fmt.Printf("Connecting to: %s\n", url)

	// Check if PocketBase is running
	if err := checkHealth(url); err != nil {
		fmt.Printf("❌ Cannot connect to PocketBase: %v\n", err)
		fmt.Println("\nPlease check:")
		fmt.Println("1. Is PocketBase running at the specified URL?")
		fmt.Println("2. Check with: curl http://192.168.100.100:8090/api/health")
		os.Exit(1)
	}

	// Check if token provided
	if token == "" {
		fmt.Println("❌ POCKETBASE_TOKEN not set")
		fmt.Println("\nPlease set:")
		fmt.Println("  export POCKETBASE_TOKEN=your_token_here")
		fmt.Println("\nTo get token:")
		fmt.Println("  curl -X POST http://192.168.100.100:8090/api/admins/auth-with-password \\")
		fmt.Println("    -H \"Content-Type: application/json\" \\")
		fmt.Println("    -d '{\"identity\":\"admin@example.com\",\"password\":\"password123\"}'")
		os.Exit(1)
	}

	fmt.Println("✅ Using POCKETBASE_TOKEN from environment")

	// Test auth first
	if err := testAuth(url, token); err != nil {
		fmt.Printf("❌ Auth test failed: %v\n", err)
		os.Exit(1)
	}

	// Create collections
	collections := []struct {
		name   string
		create func(string, string) error
	}{
		{"scanners", createScannersCollection},
		{"employees", createEmployeesCollection},
		{"attendance", createAttendanceCollection},
		{"employee_detections", createDetectionsCollection},
		{"devices", createDevicesCollection},
	}

	for _, col := range collections {
		fmt.Printf("\n📦 Creating collection: %s\n", col.name)
		if err := col.create(url, token); err != nil {
			fmt.Printf("   ⚠️  %v\n", err)
		} else {
			fmt.Printf("   ✅ Created successfully\n")
		}
	}

	fmt.Println("\n🎉 Setup complete!")
	fmt.Printf("\nAccess Admin UI: %s/_/\n", url)
}

func testAuth(baseURL, token string) error {
	url := fmt.Sprintf("%s/api/collections", baseURL)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	fmt.Println("✅ Authentication successful")
	return nil
}

func createCollection(baseURL, token, name string, fields []map[string]interface{}) error {
	// Create collection with fields using proper PocketBase format
	createURL := fmt.Sprintf("%s/api/collections", baseURL)

	createData := map[string]interface{}{
		"name":   name,
		"type":   "base",
		"fields": fields,
	}

	jsonData, _ := json.Marshal(createData)
	req, _ := http.NewRequest("POST", createURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Check if already exists
	if resp.StatusCode == http.StatusBadRequest && (bytes.Contains(body, []byte("already exists")) || bytes.Contains(body, []byte("must be unique"))) {
		// Try to update schema instead
		fmt.Printf("   Collection exists, attempting to update fields...\n")
		return updateCollectionFields(baseURL, token, name, fields)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create failed: %s - %s", resp.Status, string(body))
	}

	fmt.Printf("   Created with %d fields\n", len(fields))
	return nil
}

func updateCollectionFields(baseURL, token, name string, fields []map[string]interface{}) error {
	// Get existing collection to check current fields
	getURL := fmt.Sprintf("%s/api/collections/%s", baseURL, name)
	req, _ := http.NewRequest("GET", getURL, nil)
	req.Header.Set("Authorization", token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get collection: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var existing struct {
		ID     string                   `json:"id"`
		Fields []map[string]interface{} `json:"fields"`
	}

	if err := json.Unmarshal(body, &existing); err != nil {
		return fmt.Errorf("failed to parse collection: %v", err)
	}

	// Check which fields are missing
	existingFieldNames := make(map[string]bool)
	for _, f := range existing.Fields {
		if name, ok := f["name"].(string); ok {
			existingFieldNames[name] = true
		}
	}

	var newFields []map[string]interface{}
	for _, field := range fields {
		if name, ok := field["name"].(string); ok {
			if !existingFieldNames[name] {
				newFields = append(newFields, field)
			}
		}
	}

	if len(newFields) == 0 {
		fmt.Printf("   All fields already exist\n")
		return nil
	}

	// Update collection with new fields
	updateURL := fmt.Sprintf("%s/api/collections/%s", baseURL, name)
	updateData := map[string]interface{}{
		"fields": append(existing.Fields, newFields...),
	}

	jsonData, _ := json.Marshal(updateData)
	req, _ = http.NewRequest("PATCH", updateURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err = httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update: %v", err)
	}

	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update failed: %s - %s", resp.Status, string(body))
	}

	fmt.Printf("   Added %d new fields\n", len(newFields))
	return nil
}

func createTextField(name string, required bool) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"type":        "text",
		"required":    required,
		"unique":      false,
		"hidden":      false,
		"presentable": false,
		"system":      false,
		"options": map[string]interface{}{
			"min": 0,
			"max": 0,
		},
	}
}

func createTextFieldWithPattern(name string, required bool, pattern string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"type":        "text",
		"required":    required,
		"unique":      false,
		"hidden":      false,
		"presentable": false,
		"system":      false,
		"options": map[string]interface{}{
			"min":     0,
			"max":     0,
			"pattern": pattern,
		},
	}
}

func createNumberField(name string, required bool) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"type":        "number",
		"required":    required,
		"unique":      false,
		"hidden":      false,
		"presentable": false,
		"system":      false,
		"options": map[string]interface{}{
			"min":       nil,
			"max":       nil,
			"noDecimal": false,
		},
	}
}

func createDateField(name string, required bool) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"type":        "date",
		"required":    required,
		"unique":      false,
		"hidden":      false,
		"presentable": false,
		"system":      false,
		"options": map[string]interface{}{
			"min": "",
			"max": "",
		},
	}
}

func createBoolField(name string, required bool) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"type":        "bool",
		"required":    required,
		"unique":      false,
		"hidden":      false,
		"presentable": false,
		"system":      false,
		"options":     map[string]interface{}{},
	}
}

func createScannersCollection(baseURL, token string) error {
	fields := []map[string]interface{}{
		createTextFieldWithPattern("scanner_mac", true, "^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"),
		createDateField("last_seen", true),
	}
	return createCollection(baseURL, token, "scanners", fields)
}

func createEmployeesCollection(baseURL, token string) error {
	fields := []map[string]interface{}{
		createTextFieldWithPattern("mac_address", true, "^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"),
		createNumberField("telegram_chat_id", true),
		createTextField("name", true),
		createTextField("employee_code", false),
		createTextField("department", false),
		createTextFieldWithPattern("work_start_time", false, "^([0-1]?[0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$"),
		createBoolField("is_active", false),
	}
	return createCollection(baseURL, token, "employees", fields)
}

func createAttendanceCollection(baseURL, token string) error {
	fields := []map[string]interface{}{
		createNumberField("employee_id", true),
		createDateField("check_in_time", true),
		createDateField("check_out_time", false),
		createTextField("scanner_mac", false),
		createTextField("status", true),
		createDateField("created_date", true),
	}
	return createCollection(baseURL, token, "attendance", fields)
}

func createDetectionsCollection(baseURL, token string) error {
	fields := []map[string]interface{}{
		createNumberField("employee_id", true),
		createTextField("mac_address", true),
		createTextField("scanner_mac", true),
		createNumberField("rssi", true),
		createTextField("device_type", false),
		createBoolField("is_itag03", false),
		createDateField("detected_at", true),
	}
	return createCollection(baseURL, token, "employee_detections", fields)
}

func createDevicesCollection(baseURL, token string) error {
	fields := []map[string]interface{}{
		createTextField("mac_address", true),
		createTextField("name", false),
		createBoolField("is_whitelisted", false),
		createNumberField("rssi", false),
		createDateField("last_seen", false),
	}
	return createCollection(baseURL, token, "devices", fields)
}

func checkHealth(baseURL string) error {
	resp, err := httpClient.Get(baseURL + "/api/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %s", resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ PocketBase is running: %s\n", string(body))
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
