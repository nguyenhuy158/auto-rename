// Database logic for auto-rename
package infrastructure

import (
	"database/sql"

	"auto-rename/internal/domain"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

// NewDatabase khởi tạo kết nối database
func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	return &Database{db: db}, nil
}

// HasOriginalName kiểm tra file đã có trong DB chưa
func (d *Database) HasOriginalName(name string) (bool, error) {
	row := d.db.QueryRow("SELECT COUNT(*) FROM file_records WHERE original_name = ?", name)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// InsertFileRecord thêm bản ghi file vào DB
func (d *Database) InsertFileRecord(record domain.FileRecord) error {
	_, err := d.db.Exec(
		`INSERT INTO file_records (original_name, new_name, file_path, file_size, file_mode, mod_time, success, error_msg, renamed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.OriginalName, record.NewName, record.FilePath, record.FileSize, record.FileMode, record.ModTime, record.Success, record.ErrorMsg, record.RenamedAt,
	)
	return err
}

// Close đóng kết nối DB
func (d *Database) Close() error {
	return d.db.Close()
}
