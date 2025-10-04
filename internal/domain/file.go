// Domain entities for auto-rename
package domain

// FileRecord định nghĩa thông tin file đã được xử lý
type FileRecord struct {
	OriginalName string
	NewName      string
	FilePath     string
	FileSize     int64
	FileMode     string
	ModTime      string
	Success      bool
	ErrorMsg     string
	RenamedAt    string
}
