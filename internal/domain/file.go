// Domain entities for auto-rename
package domain

// FileRecord định nghĩa thông tin file đã được xử lý
type FileRecord struct {
	Id           int    `json:"id"`
	OriginalName string `json:"original_name"`
	NewName      string `json:"new_name"`
	FilePath     string `json:"file_path"`
	FileSize     int64  `json:"file_size"`
	FileMode     string `json:"file_mode"`
	ModTime      string `json:"mod_time"`
	Success      bool   `json:"success"`
	ErrorMsg     string `json:"error_msg"`
	RenamedAt    string `json:"renamed_at"`
}

// CronStatus tracks cron job execution state
type CronStatus struct {
	Enabled         bool   `json:"enabled"`
	NextRun         string `json:"next_run"`
	LastRun         string `json:"last_run"`
	IsRunning       bool   `json:"is_running"`
	Directory       string `json:"directory"`
	IntervalSeconds int    `json:"interval_seconds"`
	TotalScans      int    `json:"total_scans"`
	FilesProcessed  int    `json:"files_processed"`
	FilesSkipped    int    `json:"files_skipped"`
	LastError       string `json:"last_error"`
}

// DashboardData holds all data for dashboard page rendering
type DashboardData struct {
	TotalRecords      int
	SuccessfulRenames int
	FailedRenames     int
	RecentActivity    int
	RecentOperations  []FileRecord
	CronStatus        *CronStatus
	CurrentTime       string
}

// RecordsPageData holds all data for records page rendering
type RecordsPageData struct {
	Records      []FileRecord
	TotalRecords int
	CurrentPage  int
	PageSize     int
	SearchQuery  string
	StatusFilter string
	DateFrom     string
	DateTo       string
}

// LogsPageData holds all data for logs page rendering
type LogsPageData struct {
	Records      []FileRecord
	TotalRecords int
	CurrentPage  int
	PageSize     int
	SearchQuery  string
	StatusFilter string
}

// FileEntry represents a file or directory in the browser
type FileEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsDir     bool   `json:"is_dir"`
	Size      int64  `json:"size"`
	ModTime   string `json:"mod_time"`
	CanRename bool   `json:"can_rename"`
	IsImage   bool   `json:"is_image"`
	IsVideo   bool   `json:"is_video"`
	MimeType  string `json:"mime_type"`
}

// FileBrowserData holds data for file browser page
type FileBrowserData struct {
	CurrentPath string
	Entries     []FileEntry
	BaseDir     string
	Breadcrumb  []BreadcrumbItem
}

// BreadcrumbItem represents a part of the breadcrumb navigation
type BreadcrumbItem struct {
	Name string
	Path string
}
