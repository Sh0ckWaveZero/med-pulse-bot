// Package models contains data structures for the application
package models

import (
	"time"
)

// DetectionRequest represents a BLE device detection from ESP32 scanner
type DetectionRequest struct {
	ScannerMac     string `json:"scanner_mac"`
	MacAddress     string `json:"mac_address"`
	RSSI           int    `json:"rssi"`
	DeviceType     string `json:"device_type"`
	IsITag03       bool   `json:"itag03"`
	IsTargetDevice bool   `json:"target_device"` // True if MAC or UUID matches target list
	DeviceName     string `json:"device_name"`   // Custom name for target device (e.g., "MSL AirPods Pro")
}

// Employee represents an employee in the system
type Employee struct {
	ID             string
	TelegramChatID int64
	Name           string
	MacAddress     string
	WorkStartTime  string
	IsActive       bool
}

// Attendance represents an attendance record
type Attendance struct {
	ID          string
	EmployeeID  string
	CheckInTime time.Time
	ScannerMac  string
	Status      string
	CreatedDate time.Time
}

// EmployeeDetection represents a detection record for an employee
type EmployeeDetection struct {
	ID             string
	EmployeeID     string
	MacAddress     string
	ScannerMac     string
	RSSI           int
	DeviceType     string
	IsITag03       bool
	IsTargetDevice bool   // True if matched target MAC/UUID
	DeviceName     string // Custom name for target device
	DetectedAt     time.Time
}

// Scanner represents a BLE scanner device
type Scanner struct {
	ID         string
	ScannerMac string
	LastSeen   time.Time
}
