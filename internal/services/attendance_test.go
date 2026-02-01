package services

import (
	"testing"
	"time"
)

func TestCalculateStatus(t *testing.T) {
	tests := []struct {
		name          string
		checkInTime   time.Time
		workStartTime string
		want          string
	}{
		{
			name:          "On time - exactly at work start",
			checkInTime:   time.Date(2026, 2, 1, 8, 0, 0, 0, time.Local),
			workStartTime: "08:00:00",
			want:          "ontime",
		},
		{
			name:          "On time - within grace period (5 minutes)",
			checkInTime:   time.Date(2026, 2, 1, 8, 4, 0, 0, time.Local),
			workStartTime: "08:00:00",
			want:          "ontime",
		},
		{
			name:          "Late - 1 minute after grace period",
			checkInTime:   time.Date(2026, 2, 1, 8, 6, 0, 0, time.Local),
			workStartTime: "08:00:00",
			want:          "late",
		},
		{
			name:          "Late - 30 minutes late",
			checkInTime:   time.Date(2026, 2, 1, 8, 30, 0, 0, time.Local),
			workStartTime: "08:00:00",
			want:          "late",
		},
		{
			name:          "On time - before work start",
			checkInTime:   time.Date(2026, 2, 1, 7, 45, 0, 0, time.Local),
			workStartTime: "08:00:00",
			want:          "ontime",
		},
		{
			name:          "Invalid work start time - defaults to ontime",
			checkInTime:   time.Date(2026, 2, 1, 9, 0, 0, 0, time.Local),
			workStartTime: "invalid",
			want:          "ontime",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateStatus(tt.checkInTime, tt.workStartTime)
			if got != tt.want {
				t.Errorf("calculateStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateLateStatus(t *testing.T) {
	tests := []struct {
		name          string
		checkInTime   time.Time
		workStartTime string
		want          string
	}{
		{
			name:          "Late 10 minutes",
			checkInTime:   time.Date(2026, 2, 1, 8, 10, 0, 0, time.Local),
			workStartTime: "08:00:00",
			want:          "เข้าสาย 10 นาที",
		},
		{
			name:          "Late 30 minutes",
			checkInTime:   time.Date(2026, 2, 1, 8, 30, 0, 0, time.Local),
			workStartTime: "08:00:00",
			want:          "เข้าสาย 30 นาที",
		},
		{
			name:          "Invalid time - defaults to simple message",
			checkInTime:   time.Date(2026, 2, 1, 8, 30, 0, 0, time.Local),
			workStartTime: "invalid",
			want:          "เข้าสาย",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateLateStatus(tt.checkInTime, tt.workStartTime)
			if got != tt.want {
				t.Errorf("calculateLateStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
