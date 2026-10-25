package domain

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
	Status       string `json:"status"`
	RestoredAt   string `json:"restored_at"`
}

type ScanRun struct {
	ID             int    `json:"id"`
	Source         string `json:"source"`
	Status         string `json:"status"`
	StartedAt      string `json:"started_at"`
	FinishedAt     string `json:"finished_at"`
	TotalFiles     int    `json:"total_files"`
	ProcessedFiles int    `json:"processed_files"`
	RenamedFiles   int    `json:"renamed_files"`
	DuplicateFiles int    `json:"duplicate_files"`
	Error          string `json:"error"`
}

type QuarantineAction struct {
	ID             int    `json:"id"`
	OriginalPath   string `json:"original_path"`
	QuarantinePath string `json:"quarantine_path"`
	SHA256         string `json:"sha256"`
	KeptPath       string `json:"kept_path"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
	RestoredAt     string `json:"restored_at"`
}

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

type DashboardData struct {
	TotalRecords      int
	SuccessfulRenames int
	FailedRenames     int
	RecentActivity    int
	RecentOperations  []FileRecord
	CronStatus        *CronStatus
	CurrentTime       string
}

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

type LogsPageData struct {
	Records      []FileRecord
	TotalRecords int
	CurrentPage  int
	PageSize     int
	SearchQuery  string
	StatusFilter string
}

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

type FileBrowserData struct {
	CurrentPath string
	Entries     []FileEntry
	BaseDir     string
	Breadcrumb  []BreadcrumbItem
}

type BreadcrumbItem struct {
	Name string
	Path string
}

type SettingsData struct {
	Dir                string
	DryRun             bool
	WebPort            string
	WebOnly            bool
	Cron               bool
	CronExpr           string
	RenameSubfolder    bool
	FormatSize         string
	FormatDatetime     string
	FormatTimezone     string
	FormatSizeDecimals int
	UpdatedAt          string
}
