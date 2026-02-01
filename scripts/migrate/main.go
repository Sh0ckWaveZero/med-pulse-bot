package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultPocketBaseURL = "http://192.168.100.100:8090"
)

type SchemaField struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Required bool                   `json:"required"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type Collection struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Type   string        `json:"type"`
	Fields []SchemaField `json:"fields"`
}

type Migrator struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func loadEnv() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	// Get project root (scripts directory -> project root)
	projectRoot := filepath.Dir(filepath.Dir(execPath))
	envPath := filepath.Join(projectRoot, ".env")

	// Try current directory if executable path doesn't work
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		envPath = "../.env"
	}

	file, err := os.Open(envPath)
	if err != nil {
		log.Printf("⚠️  Warning: Could not open .env file: %v", err)
		return nil // Don't fail if .env doesn't exist
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Only set if not already set
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	log.Println("📝 Loaded .env file")
	return scanner.Err()
}

func NewMigrator() *Migrator {
	// Load .env file first
	loadEnv()

	baseURL := os.Getenv("POCKETBASE_URL")
	if baseURL == "" {
		baseURL = defaultPocketBaseURL
	}

	token := os.Getenv("POCKETBASE_TOKEN")
	if token == "" {
		log.Fatal("❌ Error: POCKETBASE_TOKEN not found in environment variables")
	}

	return &Migrator{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (m *Migrator) checkConnection() error {
	log.Println("🔍 Checking PocketBase connection...")

	resp, err := m.httpClient.Get(m.baseURL + "/api/health")
	if err != nil {
		return fmt.Errorf("cannot connect to PocketBase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PocketBase health check failed: %s", resp.Status)
	}

	log.Println("✅ PocketBase is running")
	return nil
}

func (m *Migrator) checkToken() error {
	log.Println("🔐 Checking authentication token...")

	if m.token == "" {
		return fmt.Errorf("POCKETBASE_TOKEN not found in environment variables")
	}

	log.Println("✅ Token found")
	return nil
}

func (m *Migrator) getCollection(name string) (*Collection, error) {
	log.Printf("📖 Fetching %s collection...\n", name)

	req, _ := http.NewRequest("GET", m.baseURL+"/api/collections/"+name, nil)
	req.Header.Set("Authorization", m.token)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get collection: %s - %s", resp.Status, string(body))
	}

	var collection Collection
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return nil, fmt.Errorf("failed to decode collection: %w", err)
	}

	log.Println("✅ Collection found")
	return &collection, nil
}

func (m *Migrator) hasField(collection *Collection, fieldName string) bool {
	for _, field := range collection.Fields {
		if field.Name == fieldName {
			return true
		}
	}
	return false
}

func (m *Migrator) addTargetDeviceFields(collection *Collection) error {
	log.Println("🔄 Adding new fields to employee_detections...")
	log.Println("   • is_target_device (Boolean)")
	log.Println("   • device_name (Text, max 255)")

	// Check if fields already exist
	if m.hasField(collection, "is_target_device") {
		log.Println("⚠️  Field 'is_target_device' already exists. Skipping...")
	} else {
		collection.Fields = append(collection.Fields, SchemaField{
			Name:     "is_target_device",
			Type:     "bool",
			Required: false,
			Options:  map[string]interface{}{},
		})
	}

	if m.hasField(collection, "device_name") {
		log.Println("⚠️  Field 'device_name' already exists. Skipping...")
	} else {
		collection.Fields = append(collection.Fields, SchemaField{
			Name:     "device_name",
			Type:     "text",
			Required: false,
			Options: map[string]interface{}{
				"max": 255,
			},
		})
	}

	// Update collection
	jsonData, _ := json.Marshal(collection)
	req, _ := http.NewRequest("PATCH", m.baseURL+"/api/collections/"+collection.ID, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", m.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update collection: %s - %s", resp.Status, string(body))
	}

	log.Println("✅ Migration completed successfully!")
	return nil
}

func (m *Migrator) verify() error {
	log.Println("\n🧪 Verifying migration...")

	collection, err := m.getCollection("employee_detections")
	if err != nil {
		return err
	}

	if !m.hasField(collection, "is_target_device") {
		return fmt.Errorf("field 'is_target_device' not found after migration")
	}
	log.Println("✅ Field 'is_target_device' verified")

	if !m.hasField(collection, "device_name") {
		return fmt.Errorf("field 'device_name' not found after migration")
	}
	log.Println("✅ Field 'device_name' verified")

	return nil
}

func (m *Migrator) Run() error {
	log.Println("🔧 PocketBase Migration: Add Target Device Fields")
	log.Println("==================================================")
	log.Printf("📍 PocketBase URL: %s\n\n", m.baseURL)

	if err := m.checkConnection(); err != nil {
		return err
	}

	if err := m.checkToken(); err != nil {
		return err
	}

	collection, err := m.getCollection("employee_detections")
	if err != nil {
		return err
	}

	if err := m.addTargetDeviceFields(collection); err != nil {
		return err
	}

	if err := m.verify(); err != nil {
		return err
	}

	log.Println("\n🎉 Migration verified successfully!")
	log.Println("\n📋 Next Steps:")
	log.Println("   1. Restart Backend API: docker-compose restart app")
	log.Println("   2. Upload firmware to ESP32")
	log.Println("   3. Test target device detection")

	return nil
}

func main() {
	migrator := NewMigrator()

	if err := migrator.Run(); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}
}
