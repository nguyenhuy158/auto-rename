package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type FileRecord struct {
	ID           int       `json:"id"`
	OriginalName string    `json:"original_name"`
	NewName      string    `json:"new_name"`
	FilePath     string    `json:"file_path"`
	FileSize     int64     `json:"file_size"`
	FileMode     string    `json:"file_mode"`
	ModTime      time.Time `json:"mod_time"`
	RenamedAt    time.Time `json:"renamed_at"`
	Success      bool      `json:"success"`
	ErrorMsg     string    `json:"error_msg,omitempty"`
}

type Database struct {
	db *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	database := &Database{db: db}
	if err := database.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return database, nil
}

func (d *Database) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS file_renames (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		original_name TEXT NOT NULL,
		new_name TEXT NOT NULL,
		file_path TEXT NOT NULL,
		file_size INTEGER,
		file_mode TEXT,
		mod_time DATETIME,
		renamed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		success BOOLEAN DEFAULT TRUE,
		error_msg TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_original_name ON file_renames(original_name);
	CREATE INDEX IF NOT EXISTS idx_new_name ON file_renames(new_name);
	CREATE INDEX IF NOT EXISTS idx_renamed_at ON file_renames(renamed_at);
	`

	_, err := d.db.Exec(query)
	return err
}

func (d *Database) InsertFileRecord(record FileRecord) error {
	query := `
	INSERT INTO file_renames (
		original_name, new_name, file_path, file_size,
		file_mode, mod_time, renamed_at, success, error_msg
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		record.OriginalName,
		record.NewName,
		record.FilePath,
		record.FileSize,
		record.FileMode,
		record.ModTime,
		record.RenamedAt,
		record.Success,
		record.ErrorMsg,
	)

	if err != nil {
		return fmt.Errorf("failed to insert record: %w", err)
	}

	return nil
}

func (d *Database) GetAllRecords() ([]FileRecord, error) {
	query := `
	SELECT id, original_name, new_name, file_path, file_size,
		   file_mode, mod_time, renamed_at, success, error_msg
	FROM file_renames
	ORDER BY renamed_at DESC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var records []FileRecord
	for rows.Next() {
		var record FileRecord
		var errorMsg sql.NullString

		err := rows.Scan(
			&record.ID,
			&record.OriginalName,
			&record.NewName,
			&record.FilePath,
			&record.FileSize,
			&record.FileMode,
			&record.ModTime,
			&record.RenamedAt,
			&record.Success,
			&errorMsg,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if errorMsg.Valid {
			record.ErrorMsg = errorMsg.String
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return records, nil
}

func (d *Database) GetRecordsByOriginalName(originalName string) ([]FileRecord, error) {
	query := `
	SELECT id, original_name, new_name, file_path, file_size,
		   file_mode, mod_time, renamed_at, success, error_msg
	FROM file_renames
	WHERE original_name LIKE ?
	ORDER BY renamed_at DESC
	`

	rows, err := d.db.Query(query, "%"+originalName+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var records []FileRecord
	for rows.Next() {
		var record FileRecord
		var errorMsg sql.NullString

		err := rows.Scan(
			&record.ID,
			&record.OriginalName,
			&record.NewName,
			&record.FilePath,
			&record.FileSize,
			&record.FileMode,
			&record.ModTime,
			&record.RenamedAt,
			&record.Success,
			&errorMsg,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if errorMsg.Valid {
			record.ErrorMsg = errorMsg.String
		}

		records = append(records, record)
	}

	return records, nil
}

func (d *Database) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total records
	var total int
	err := d.db.QueryRow("SELECT COUNT(*) FROM file_renames").Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total_records"] = total

	// Successful renames
	var successful int
	err = d.db.QueryRow("SELECT COUNT(*) FROM file_renames WHERE success = TRUE").Scan(&successful)
	if err != nil {
		return nil, err
	}
	stats["successful_renames"] = successful

	// Failed renames
	var failed int
	err = d.db.QueryRow("SELECT COUNT(*) FROM file_renames WHERE success = FALSE").Scan(&failed)
	if err != nil {
		return nil, err
	}
	stats["failed_renames"] = failed

	// Recent activity (last 24 hours)
	var recent int
	err = d.db.QueryRow("SELECT COUNT(*) FROM file_renames WHERE renamed_at > datetime('now', '-1 day')").Scan(&recent)
	if err != nil {
		return nil, err
	}
	stats["recent_activity"] = recent

	return stats, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

// HasOriginalName returns true if a record with the given original name already exists.
func (d *Database) HasOriginalName(name string) (bool, error) {
	query := `SELECT 1 FROM file_renames WHERE original_name = ? LIMIT 1`
	row := d.db.QueryRow(query, name)
	var dummy int
	err := row.Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("has original name query failed: %w", err)
	}
	return true, nil
}

func getFileInfo(filePath string) (int64, string, time.Time, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, "", time.Time{}, err
	}

	return info.Size(), info.Mode().String(), info.ModTime(), nil
}
