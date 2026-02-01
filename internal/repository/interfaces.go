// Package repository defines repository interfaces for data access
package repository

import (
	"context"
	"telegram-bot-med/internal/models"
)

// EmployeeRepository defines the interface for employee data access
type EmployeeRepository interface {
	// GetByMacAddress retrieves an employee by their MAC address
	GetByMacAddress(ctx context.Context, macAddress string) (*models.Employee, error)
	// IsCheckedInToday checks if employee already checked in today
	IsCheckedInToday(ctx context.Context, employeeID string) (bool, error)
}

// AttendanceRepository defines the interface for attendance data access
type AttendanceRepository interface {
	// Create records a new attendance check-in
	Create(ctx context.Context, attendance *models.Attendance) error
}

// EmployeeDetectionRepository defines the interface for employee detection data access
type EmployeeDetectionRepository interface {
	// Create saves a new employee detection record
	Create(ctx context.Context, detection *models.EmployeeDetection) error
}

// ScannerRepository defines the interface for scanner data access
type ScannerRepository interface {
	// UpdateActivity updates the last seen timestamp for a scanner
	UpdateActivity(ctx context.Context, scannerMac string) error
}
