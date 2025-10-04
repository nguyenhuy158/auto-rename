package main

import (
	"sync"
	"time"
)

// CronStatus tracks the current status of the cron job
type CronStatus struct {
	mu             sync.RWMutex
	Enabled        bool      `json:"enabled"`
	LastRun        time.Time `json:"last_run"`
	NextRun        time.Time `json:"next_run"`
	Directory      string    `json:"directory"`
	Interval       int       `json:"interval_seconds"` // in seconds
	IsRunning      bool      `json:"is_running"`
	LastError      string    `json:"last_error,omitempty"`
	TotalScans     int       `json:"total_scans"`
	FilesProcessed int       `json:"files_processed"`
	FilesSkipped   int       `json:"files_skipped"`
}

// Global cron status instance
var globalCronStatus = &CronStatus{
	Enabled:  false,
	Interval: 60, // default 60 seconds
}

// GetCronStatus returns a copy of the current cron status
func GetCronStatus() CronStatus {
	globalCronStatus.mu.RLock()
	defer globalCronStatus.mu.RUnlock()
	return *globalCronStatus
}

// UpdateCronStatus updates the cron status with new information
func UpdateCronStatus(updates func(*CronStatus)) {
	globalCronStatus.mu.Lock()
	defer globalCronStatus.mu.Unlock()
	updates(globalCronStatus)
}

// InitializeCronStatus sets up the initial cron status
func InitializeCronStatus(enabled bool, directory string) {
	UpdateCronStatus(func(status *CronStatus) {
		status.Enabled = enabled
		status.Directory = directory
		if enabled {
			status.NextRun = time.Now().Add(time.Duration(status.Interval) * time.Second)
		}
	})
}

// MarkCronStart marks the beginning of a cron scan
func MarkCronStart() {
	UpdateCronStatus(func(status *CronStatus) {
		status.IsRunning = true
		status.LastRun = time.Now()
		status.TotalScans++
	})
}

// MarkCronComplete marks the completion of a cron scan
func MarkCronComplete(processed, skipped int, err error) {
	UpdateCronStatus(func(status *CronStatus) {
		status.IsRunning = false
		status.FilesProcessed += processed
		status.FilesSkipped += skipped
		status.NextRun = time.Now().Add(time.Duration(status.Interval) * time.Second)
		if err != nil {
			status.LastError = err.Error()
		} else {
			status.LastError = ""
		}
	})
}
