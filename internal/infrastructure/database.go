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
	// Auto-create table if not exists
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS file_records (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        original_name TEXT,
        new_name TEXT,
        file_path TEXT,
        file_size INTEGER,
        file_mode TEXT,
        mod_time TEXT,
        success BOOLEAN,
        error_msg TEXT,
        renamed_at TEXT
    );`
	if _, err := db.Exec(createTableSQL); err != nil {
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

func (d *Database) GetAllFileRecords() ([]domain.FileRecord, error) {
	rows, err := d.db.Query("SELECT original_name, new_name, file_path, file_size, file_mode, mod_time, success, error_msg, renamed_at, id FROM file_records ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []domain.FileRecord
	for rows.Next() {
		var r domain.FileRecord
		err := rows.Scan(&r.OriginalName, &r.NewName, &r.FilePath, &r.FileSize, &r.FileMode, &r.ModTime, &r.Success, &r.ErrorMsg, &r.RenamedAt, &r.Id)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// Close đóng kết nối DB
func (d *Database) Close() error {
	return d.db.Close()
}
