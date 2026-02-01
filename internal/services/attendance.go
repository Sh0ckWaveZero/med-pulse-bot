// Package services implements business logic for the application
package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"med-pulse-bot/internal/models"
	"med-pulse-bot/internal/repository"
)

// AttendanceProcessor defines the interface for attendance processing
type AttendanceProcessor interface {
	ProcessDetection(ctx context.Context, req *models.DetectionRequest) error
}

// AttendanceService handles attendance business logic
type AttendanceService struct {
	employeeRepo   repository.EmployeeRepository
	attendanceRepo repository.AttendanceRepository
	detectionRepo  repository.EmployeeDetectionRepository
	scannerRepo    repository.ScannerRepository
	botNotifier    BotNotifier
}

// BotNotifier defines the interface for bot notifications
type BotNotifier interface {
	SendNotification(message string)
	SendPersonalNotification(chatID int64, message string)
}

// NewAttendanceService creates a new attendance service
func NewAttendanceService(
	employeeRepo repository.EmployeeRepository,
	attendanceRepo repository.AttendanceRepository,
	detectionRepo repository.EmployeeDetectionRepository,
	scannerRepo repository.ScannerRepository,
	botNotifier BotNotifier,
) *AttendanceService {
	return &AttendanceService{
		employeeRepo:   employeeRepo,
		attendanceRepo: attendanceRepo,
		detectionRepo:  detectionRepo,
		scannerRepo:    scannerRepo,
		botNotifier:    botNotifier,
	}
}

// ProcessDetection processes a BLE device detection
func (s *AttendanceService) ProcessDetection(ctx context.Context, req *models.DetectionRequest) error {
	// Update scanner activity (optional - comment out if not needed)
	// if err := s.scannerRepo.UpdateActivity(ctx, req.ScannerMac); err != nil {
	// 	log.Printf("Warning: failed to update scanner activity: %v", err)
	// }

	// Check if MAC/UUID matches any employee (target device detection)
	employee, err := s.employeeRepo.GetByMacAddress(ctx, req.MacAddress)
	if err != nil {
		// Not a registered employee device - ignore silently
		return nil
	}

	// Device belongs to an employee - mark as target device
	log.Printf("🎯 TARGET DEVICE detected: Employee=%s, MAC=%s, RSSI=%d",
		employee.Name, req.MacAddress, req.RSSI)
	req.IsTargetDevice = true
	req.DeviceName = employee.Name

	// Check if device is close enough (RSSI threshold for ~10 meters)
	const rssiThreshold = -70
	if req.RSSI < rssiThreshold {
		log.Printf("Device %s too far (RSSI: %d, need: %d or higher)", req.MacAddress, req.RSSI, rssiThreshold)
		return nil
	}

	// Check if already checked in today
	isCheckedIn, err := s.employeeRepo.IsCheckedInToday(ctx, employee.ID)
	if err != nil {
		return fmt.Errorf("failed to check attendance status: %w", err)
	}

	// If not checked in, save detection and record attendance
	if !isCheckedIn {
		if err := s.saveDetection(ctx, employee.ID, req); err != nil {
			return fmt.Errorf("failed to save detection: %w", err)
		}

		if err := s.recordAttendance(ctx, employee, req.ScannerMac); err != nil {
			return fmt.Errorf("failed to record attendance: %w", err)
		}
	}

	return nil
}

// saveDetection saves the detection record
func (s *AttendanceService) saveDetection(ctx context.Context, employeeID string, req *models.DetectionRequest) error {
	detection := &models.EmployeeDetection{
		EmployeeID:     employeeID,
		MacAddress:     req.MacAddress,
		ScannerMac:     req.ScannerMac,
		RSSI:           req.RSSI,
		DeviceType:     req.DeviceType,
		IsITag03:       req.IsITag03,
		IsTargetDevice: req.IsTargetDevice,
		DeviceName:     req.DeviceName,
		DetectedAt:     time.Now(),
	}

	if err := s.detectionRepo.Create(ctx, detection); err != nil {
		return fmt.Errorf("failed to create detection: %w", err)
	}

	if req.IsTargetDevice {
		log.Printf("💾 Saved TARGET DEVICE detection: Employee=%s, Device=%s, MAC=%s, RSSI=%d",
			employeeID, req.DeviceName, req.MacAddress, req.RSSI)
	} else {
		log.Printf("💾 Saved detection for employee ID %s: MAC=%s, RSSI=%d, Type=%s",
			employeeID, req.MacAddress, req.RSSI, req.DeviceType)
	}

	return nil
}

// recordAttendance records attendance and sends notifications
func (s *AttendanceService) recordAttendance(ctx context.Context, employee *models.Employee, scannerMac string) error {
	now := time.Now()
	status := calculateStatus(now, employee.WorkStartTime)

	attendance := &models.Attendance{
		EmployeeID:  employee.ID,
		CheckInTime: now,
		ScannerMac:  scannerMac,
		Status:      status,
		CreatedDate: now,
	}

	if err := s.attendanceRepo.Create(ctx, attendance); err != nil {
		return fmt.Errorf("failed to create attendance record: %w", err)
	}

	log.Printf("✅ Employee %s checked in at %s (Status: %s)",
		employee.Name, now.Format("15:04:05"), status)

	// Send notification to employee
	s.sendCheckInNotification(employee, now, scannerMac, status)

	return nil
}

// sendCheckInNotification sends check-in notification to employee
func (s *AttendanceService) sendCheckInNotification(employee *models.Employee, checkInTime time.Time, scannerMac, status string) {
	statusEmoji := "✅"
	statusText := "เข้างานตรงเวลา"

	if status == "late" {
		statusEmoji = "⚠️"
		statusText = calculateLateStatus(checkInTime, employee.WorkStartTime)
	}

	message := fmt.Sprintf(
		"%s *สวัสดีตอนเช้า คุณ%s!*\n\n"+
			"🕐 เวลาเข้างาน: `%s`\n"+
			"📍 สถานที่: `Scanner %s`\n"+
			"⏰ สถานะ: *%s*\n\n"+
			"ขอให้มีความสุขกับการทำงานวันนี้! 😊",
		statusEmoji, employee.Name, checkInTime.Format("15:04:05"), scannerMac, statusText,
	)

	s.botNotifier.SendPersonalNotification(employee.TelegramChatID, message)

	// Send to admin if late
	if status == "late" {
		adminMessage := fmt.Sprintf("⚠️ *พนักงานเข้าสาย*\n👤 ชื่อ: `%s`\n🕐 เวลา: `%s`\n⏰ %s",
			employee.Name, checkInTime.Format("15:04:05"), statusText)
		s.botNotifier.SendNotification(adminMessage)
	}
}

// calculateStatus determines if check-in is on time or late
func calculateStatus(checkInTime time.Time, workStartTime string) string {
	workStart, err := time.Parse("15:04:05", workStartTime)
	if err != nil {
		return "ontime" // Default to ontime if can't parse
	}

	todayWorkStart := time.Date(
		checkInTime.Year(),
		checkInTime.Month(),
		checkInTime.Day(),
		workStart.Hour(),
		workStart.Minute(),
		workStart.Second(),
		0,
		checkInTime.Location(),
	)

	// Grace period: 5 minutes
	gracePeriod := 5 * time.Minute

	if checkInTime.Before(todayWorkStart.Add(gracePeriod)) {
		return "ontime"
	}

	return "late"
}

// calculateLateStatus calculates late minutes for display
func calculateLateStatus(checkInTime time.Time, workStartTime string) string {
	workStart, err := time.Parse("15:04:05", workStartTime)
	if err != nil {
		return "เข้าสาย"
	}

	todayWorkStart := time.Date(
		checkInTime.Year(),
		checkInTime.Month(),
		checkInTime.Day(),
		workStart.Hour(),
		workStart.Minute(),
		workStart.Second(),
		0,
		checkInTime.Location(),
	)

	lateMinutes := int(checkInTime.Sub(todayWorkStart).Minutes())
	return fmt.Sprintf("เข้าสาย %d นาที", lateMinutes)
}
