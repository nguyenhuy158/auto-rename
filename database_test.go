package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabase(t *testing.T) {
	// Create a temporary database file
	dbPath := "./test_renames.db"
	defer os.Remove(dbPath)

	// Initialize database
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test inserting a record
	record := FileRecord{
		OriginalName: "test.txt",
		NewName:      "550e8400-e29b-41d4-a716-446655440000.txt",
		FilePath:     "/tmp",
		FileSize:     1024,
		FileMode:     "-rw-r--r--",
		ModTime:      time.Now(),
		RenamedAt:    time.Now(),
		Success:      true,
		ErrorMsg:     "",
	}

	err = db.InsertFileRecord(record)
	if err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	// Test getting all records
	records, err := db.GetAllRecords()
	if err != nil {
		t.Fatalf("Failed to get records: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	if records[0].OriginalName != "test.txt" {
		t.Errorf("Expected original name 'test.txt', got '%s'", records[0].OriginalName)
	}

	// Test search functionality
	searchRecords, err := db.GetRecordsByOriginalName("test")
	if err != nil {
		t.Fatalf("Failed to search records: %v", err)
	}

	if len(searchRecords) != 1 {
		t.Fatalf("Expected 1 search result, got %d", len(searchRecords))
	}

	// Test stats
	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats["total_records"] != 1 {
		t.Errorf("Expected total_records to be 1, got %v", stats["total_records"])
	}

	if stats["successful_renames"] != 1 {
		t.Errorf("Expected successful_renames to be 1, got %v", stats["successful_renames"])
	}
}

func TestFileInfo(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	content := []byte("Hello, World!")
	err := os.WriteFile(tmpFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test getFileInfo function
	size, mode, modTime, err := getFileInfo(tmpFile)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("Expected file size %d, got %d", len(content), size)
	}

	if mode != "-rw-r--r--" {
		t.Errorf("Expected file mode '-rw-r--r--', got '%s'", mode)
	}

	if modTime.IsZero() {
		t.Error("Expected non-zero modification time")
	}
}

func TestIntegrationRenameWithDatabase(t *testing.T) {
	// Create temporary directory and files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{"file1.txt", "file2.jpg", "document.pdf"}
	for _, filename := range testFiles {
		content := []byte("test content")
		err := os.WriteFile(filepath.Join(tmpDir, filename), content, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create database
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test dry run
	config := Config{
		Dir:    tmpDir,
		DryRun: true,
		DbPath: dbPath,
	}

	err = renameFiles(config, db)
	if err != nil {
		t.Fatalf("Failed to run dry rename: %v", err)
	}

	// Check that files still exist with original names
	for _, filename := range testFiles {
		if _, err := os.Stat(filepath.Join(tmpDir, filename)); os.IsNotExist(err) {
			t.Errorf("File %s should still exist after dry run", filename)
		}
	}

	// Check that records were created
	records, err := db.GetAllRecords()
	if err != nil {
		t.Fatalf("Failed to get records: %v", err)
	}

	if len(records) != len(testFiles) {
		t.Errorf("Expected %d records, got %d", len(testFiles), len(records))
	}

	// Test actual rename
	config.DryRun = false
	err = renameFiles(config, db)
	if err != nil {
		t.Fatalf("Failed to run actual rename: %v", err)
	}

	// Check that original files no longer exist
	for _, filename := range testFiles {
		if _, err := os.Stat(filepath.Join(tmpDir, filename)); !os.IsNotExist(err) {
			t.Errorf("Original file %s should not exist after rename", filename)
		}
	}

	// Check that we have double the records now
	records, err = db.GetAllRecords()
	if err != nil {
		t.Fatalf("Failed to get records after rename: %v", err)
	}

	if len(records) != len(testFiles)*2 {
		t.Errorf("Expected %d records after both runs, got %d", len(testFiles)*2, len(records))
	}
}
