package main

import (
	"sync"
	"time"
)

type CronRunLog struct {
	Timestamp time.Time `json:"timestamp"`
	Processed int       `json:"processed"`
	Skipped   int       `json:"skipped"`
	Error     string    `json:"error,omitempty"`
}

type CronStatus struct {
	mu             sync.RWMutex
	Enabled        bool         `json:"enabled"`
	LastRun        time.Time    `json:"last_run"`
	NextRun        time.Time    `json:"next_run"`
	Directory      string       `json:"directory"`
	Interval       int          `json:"interval_seconds"` // in seconds
	IsRunning      bool         `json:"is_running"`
	LastError      string       `json:"last_error,omitempty"`
	TotalScans     int          `json:"total_scans"`
	FilesProcessed int          `json:"files_processed"`
	FilesSkipped   int          `json:"files_skipped"`
	RunLogs        []CronRunLog `json:"run_logs"`
}

// Global cron status instance
var globalCronStatus = &CronStatus{
	Enabled:  false,
	Interval: 60, // default 60 seconds
	RunLogs:  make([]CronRunLog, 0),
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
		// Append to run logs
		runLog := CronRunLog{
			Timestamp: time.Now(),
			Processed: processed,
			Skipped:   skipped,
		}
		if err != nil {
			runLog.Error = err.Error()
		}
		status.RunLogs = append(status.RunLogs, runLog)
		// Limit log size to last 100 runs
		if len(status.RunLogs) > 100 {
			status.RunLogs = status.RunLogs[len(status.RunLogs)-100:]
		}
	})
}
