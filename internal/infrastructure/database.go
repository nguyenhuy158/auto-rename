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

func (d *Database) GetFileRecordsPage(page, pageSize int) ([]domain.FileRecord, error) {
	offset := (page - 1) * pageSize
	rows, err := d.db.Query(
		"SELECT original_name, new_name, file_path, file_size, file_mode, mod_time, success, error_msg, renamed_at, id FROM file_records ORDER BY id DESC LIMIT ? OFFSET ?",
		pageSize, offset,
	)
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

func (d *Database) CountFileRecords() (int, error) {
	row := d.db.QueryRow("SELECT COUNT(*) FROM file_records")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetTotalRecords returns total count of all records
func (d *Database) GetTotalRecords() (int, error) {
	return d.CountFileRecords()
}

// GetSuccessfulRenames returns count of successful rename operations
func (d *Database) GetSuccessfulRenames() (int, error) {
	row := d.db.QueryRow("SELECT COUNT(*) FROM file_records WHERE success = 1")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetFailedRenames returns count of failed rename operations
func (d *Database) GetFailedRenames() (int, error) {
	row := d.db.QueryRow("SELECT COUNT(*) FROM file_records WHERE success = 0")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetRecentActivity returns count of operations in last N hours
func (d *Database) GetRecentActivity(hours int) (int, error) {
	query := `SELECT COUNT(*) FROM file_records 
	WHERE datetime(renamed_at) > datetime('now', '-' || ? || ' hours')`
	row := d.db.QueryRow(query, hours)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetRecentOperations returns latest N file operations with full details
func (d *Database) GetRecentOperations(limit int) ([]domain.FileRecord, error) {
	rows, err := d.db.Query(
		"SELECT id, original_name, new_name, file_path, file_size, file_mode, mod_time, success, error_msg, renamed_at FROM file_records ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []domain.FileRecord
	for rows.Next() {
		var r domain.FileRecord
		err := rows.Scan(&r.Id, &r.OriginalName, &r.NewName, &r.FilePath, &r.FileSize, &r.FileMode, &r.ModTime, &r.Success, &r.ErrorMsg, &r.RenamedAt)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// GetRecordsByFilter returns paginated and filtered records for records page
func (d *Database) GetRecordsByFilter(page, pageSize int, search, status string, dateFrom, dateTo string) ([]domain.FileRecord, error) {
	offset := (page - 1) * pageSize
	query := "SELECT id, original_name, new_name, file_path, file_size, file_mode, mod_time, success, error_msg, renamed_at FROM file_records WHERE 1=1"
	var args []interface{}

	// Add search filter
	if search != "" {
		query += " AND (original_name LIKE ? OR new_name LIKE ?)"
		searchParam := "%" + search + "%"
		args = append(args, searchParam, searchParam)
	}

	// Add status filter
	if status == "success" {
		query += " AND success = 1"
	} else if status == "failed" {
		query += " AND success = 0"
	}

	// Add date range filter
	if dateFrom != "" {
		query += " AND datetime(renamed_at) >= datetime(?)"
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		query += " AND datetime(renamed_at) <= datetime(?)"
		args = append(args, dateTo)
	}

	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []domain.FileRecord
	for rows.Next() {
		var r domain.FileRecord
		err := rows.Scan(&r.Id, &r.OriginalName, &r.NewName, &r.FilePath, &r.FileSize, &r.FileMode, &r.ModTime, &r.Success, &r.ErrorMsg, &r.RenamedAt)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// GetRecordCountByFilter returns total count of records matching filters
func (d *Database) GetRecordCountByFilter(search, status string, dateFrom, dateTo string) (int, error) {
	query := "SELECT COUNT(*) FROM file_records WHERE 1=1"
	var args []interface{}

	if search != "" {
		query += " AND (original_name LIKE ? OR new_name LIKE ?)"
		searchParam := "%" + search + "%"
		args = append(args, searchParam, searchParam)
	}

	if status == "success" {
		query += " AND success = 1"
	} else if status == "failed" {
		query += " AND success = 0"
	}

	if dateFrom != "" {
		query += " AND datetime(renamed_at) >= datetime(?)"
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		query += " AND datetime(renamed_at) <= datetime(?)"
		args = append(args, dateTo)
	}

	row := d.db.QueryRow(query, args...)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Close đóng kết nối DB
func (d *Database) Close() error {
	return d.db.Close()
}
