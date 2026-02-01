package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"med-pulse-bot/internal/models"
	"med-pulse-bot/internal/services"
)

// mockAttendanceService is a mock implementation for testing
type mockAttendanceService struct {
	processDetectionCalled bool
	lastRequest            *models.DetectionRequest
	returnError            error
}

func (m *mockAttendanceService) ProcessDetection(ctx context.Context, req *models.DetectionRequest) error {
	m.processDetectionCalled = true
	m.lastRequest = req
	return m.returnError
}

// Ensure mock implements the interface
var _ services.AttendanceProcessor = (*mockAttendanceService)(nil)

func TestHandleDetect(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           interface{}
		wantStatusCode int
		wantCalled     bool
	}{
		{
			name:   "Valid detection request",
			method: http.MethodPost,
			body: models.DetectionRequest{
				ScannerMac: "AA:BB:CC:DD:EE:FF",
				MacAddress: "11:22:33:44:55:66",
				RSSI:       -50,
				DeviceType: "iTag03",
				IsITag03:   true,
			},
			wantStatusCode: http.StatusOK,
			wantCalled:     true,
		},
		{
			name:           "Invalid method - GET",
			method:         http.MethodGet,
			body:           nil,
			wantStatusCode: http.StatusMethodNotAllowed,
			wantCalled:     false,
		},
		{
			name:           "Invalid JSON body",
			method:         http.MethodPost,
			body:           "invalid json",
			wantStatusCode: http.StatusBadRequest,
			wantCalled:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := &mockAttendanceService{}
			handler := NewDetectionHandler(mockService)

			// Prepare request body
			var bodyBytes []byte
			var err error
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					bodyBytes = []byte(str)
				} else {
					bodyBytes, err = json.Marshal(tt.body)
					if err != nil {
						t.Fatalf("Failed to marshal body: %v", err)
					}
				}
			}

			// Create request
			req := httptest.NewRequest(tt.method, "/api/detect", bytes.NewReader(bodyBytes))
			if tt.body != nil && tt.body != "invalid json" {
				req.Header.Set("Content-Type", "application/json")
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handler.HandleDetect(rr, req)

			// Check status code
			if rr.Code != tt.wantStatusCode {
				t.Errorf("HandleDetect() status = %v, want %v", rr.Code, tt.wantStatusCode)
			}

			// Check if service was called
			if mockService.processDetectionCalled != tt.wantCalled {
				t.Errorf("ProcessDetection called = %v, want %v", mockService.processDetectionCalled, tt.wantCalled)
			}

			// Verify request was passed correctly
			if tt.wantCalled && mockService.lastRequest != nil {
				if req, ok := tt.body.(models.DetectionRequest); ok {
					if mockService.lastRequest.MacAddress != req.MacAddress {
						t.Errorf("MacAddress = %v, want %v", mockService.lastRequest.MacAddress, req.MacAddress)
					}
				}
			}
		})
	}
}
