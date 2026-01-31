// Cron state tracking for monitoring
package infrastructure

import (
	"auto-rename/internal/domain"
	"sync"
	"time"
)

// CronStateManager tracks cron job execution state
var (
	cronStateMutex sync.RWMutex
	cronState      = &domain.CronStatus{
		Enabled:         false,
		NextRun:         time.Time{}.Format(time.RFC3339),
		LastRun:         time.Time{}.Format(time.RFC3339),
		IsRunning:       false,
		Directory:       "",
		IntervalSeconds: 60,
		TotalScans:      0,
		FilesProcessed:  0,
		FilesSkipped:    0,
		LastError:       "",
	}
)

// SetCronEnabled sets whether cron is enabled
func SetCronEnabled(enabled bool, directory string, intervalSeconds int) {
	cronStateMutex.Lock()
	defer cronStateMutex.Unlock()
	cronState.Enabled = enabled
	cronState.Directory = directory
	cronState.IntervalSeconds = intervalSeconds
	if enabled {
		cronState.NextRun = time.Now().Add(time.Duration(intervalSeconds) * time.Second).Format(time.RFC3339)
	}
}

// SetCronRunning sets whether cron is currently executing
func SetCronRunning(running bool) {
	cronStateMutex.Lock()
	defer cronStateMutex.Unlock()
	cronState.IsRunning = running
}

// RecordCronExecution records metrics after a cron scan
func RecordCronExecution(nextRunTime time.Time, filesProcessed, filesSkipped int, err error) {
	cronStateMutex.Lock()
	defer cronStateMutex.Unlock()
	cronState.LastRun = time.Now().Format(time.RFC3339)
	cronState.NextRun = nextRunTime.Format(time.RFC3339)
	cronState.TotalScans++
	cronState.FilesProcessed += filesProcessed
	cronState.FilesSkipped += filesSkipped
	cronState.IsRunning = false
	if err != nil {
		cronState.LastError = err.Error()
	} else {
		cronState.LastError = ""
	}
}

// GetCronStatus returns current cron status (thread-safe copy)
func GetCronStatus() *domain.CronStatus {
	cronStateMutex.RLock()
	defer cronStateMutex.RUnlock()

	// Return a copy to prevent external mutations
	status := *cronState
	return &status
}
