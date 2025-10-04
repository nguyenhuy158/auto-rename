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
