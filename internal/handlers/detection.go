// Package handlers provides HTTP handlers for API endpoints
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"med-pulse-bot/internal/models"
	"med-pulse-bot/internal/services"
)

// DetectionHandler handles BLE device detection requests
type DetectionHandler struct {
	service services.AttendanceProcessor
}

// NewDetectionHandler creates a new detection handler
func NewDetectionHandler(service services.AttendanceProcessor) *DetectionHandler {
	return &DetectionHandler{service: service}
}

// HandleDetect processes BLE scanner detection requests
func (h *DetectionHandler) HandleDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.DetectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log detection with target device info
	if req.IsTargetDevice {
		log.Printf("🎯 [TARGET DEVICE] Scanner: %s | Device: %s | MAC: %s | RSSI: %d | Type: %s",
			req.ScannerMac, req.DeviceName, req.MacAddress, req.RSSI, req.DeviceType)
	} else {
		log.Printf("[Scanner: %s] Detected MAC: %s, RSSI: %d, Type: %s, iTag03: %v",
			req.ScannerMac, req.MacAddress, req.RSSI, req.DeviceType, req.IsITag03)
	}

	if req.IsITag03 {
		log.Printf("🏷️  iTag03 detected from scanner %s: MAC=%s, RSSI=%d",
			req.ScannerMac, req.MacAddress, req.RSSI)
	}

	// Process detection with request context
	ctx := r.Context()
	if err := h.service.ProcessDetection(ctx, &req); err != nil {
		log.Printf("Error processing detection: %v", err)
		// Don't return error to client - detection is async
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
