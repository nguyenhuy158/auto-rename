package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	Dir     string
	DryRun  bool
	WebPort string
	WebOnly bool
	DbPath  string
}

func main() {
	config := parseFlags()

	if err := validateConfig(config); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize database
	db, err := NewDatabase(config.DbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// If web-only mode, start web server and exit
	if config.WebOnly {
		webServer := NewWebServer(db, config.WebPort)
		log.Fatal(webServer.Start())
		return
	}

	// Rename files
	if err := renameFiles(config, db); err != nil {
		log.Fatalf("Error renaming files: %v", err)
	}

	// Start web server in background if port is specified
	if config.WebPort != "" {
		fmt.Printf("\nStarting web server on port %s...\n", config.WebPort)
		fmt.Printf("View results at: http://localhost:%s\n", config.WebPort)

		webServer := NewWebServer(db, config.WebPort)
		log.Fatal(webServer.Start())
	}
}

func parseFlags() Config {
	var config Config
	flag.StringVar(&config.Dir, "dir", "", "Directory containing files to rename (required)")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Preview changes without actually renaming files")
	flag.StringVar(&config.WebPort, "web-port", "8080", "Port for web interface (default: 8080)")
	flag.BoolVar(&config.WebOnly, "web-only", false, "Only start web server without renaming files")
	flag.StringVar(&config.DbPath, "db", "./file_renames.db", "SQLite database path")
	flag.Parse()

	return config
}

func validateConfig(config Config) error {
	// If web-only mode, don't require directory
	if config.WebOnly {
		return nil
	}

	if config.Dir == "" {
		return fmt.Errorf("directory path is required. Use -dir flag")
	}

	if _, err := os.Stat(config.Dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", config.Dir)
	}

	return nil
}

func renameFiles(config Config, db *Database) error {
	fmt.Printf("Scanning directory: %s\n", config.Dir)
	if config.DryRun {
		fmt.Println("DRY RUN MODE - No files will be renamed")
	}

	files, err := os.ReadDir(config.Dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	renamedCount := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		oldPath := filepath.Join(config.Dir, file.Name())
		newName := generateUUIDName(file.Name())
		newPath := filepath.Join(config.Dir, newName)

		// Get file info before renaming
		fileSize, fileMode, modTime, err := getFileInfo(oldPath)
		if err != nil {
			log.Printf("Failed to get file info for %s: %v", file.Name(), err)
			// Record the failure in database
			record := FileRecord{
				OriginalName: file.Name(),
				NewName:      newName,
				FilePath:     config.Dir,
				Success:      false,
				ErrorMsg:     fmt.Sprintf("Failed to get file info: %v", err),
				RenamedAt:    time.Now(),
			}
			if dbErr := db.InsertFileRecord(record); dbErr != nil {
				log.Printf("Failed to record error in database: %v", dbErr)
			}
			continue
		}

		fmt.Printf("  %s -> %s\n", file.Name(), newName)

		var renameErr error
		success := true
		errorMsg := ""

		if !config.DryRun {
			renameErr = os.Rename(oldPath, newPath)
			if renameErr != nil {
				success = false
				errorMsg = renameErr.Error()
				log.Printf("Failed to rename %s: %v", file.Name(), renameErr)
			}
		}

		// Record the operation in database
		record := FileRecord{
			OriginalName: file.Name(),
			NewName:      newName,
			FilePath:     config.Dir,
			FileSize:     fileSize,
			FileMode:     fileMode,
			ModTime:      modTime,
			Success:      success,
			ErrorMsg:     errorMsg,
			RenamedAt:    time.Now(),
		}

		if dbErr := db.InsertFileRecord(record); dbErr != nil {
			log.Printf("Failed to record operation in database: %v", dbErr)
		}

		if success {
			renamedCount++
		}
	}

	if config.DryRun {
		fmt.Printf("\nWould rename %d files\n", renamedCount)
	} else {
		fmt.Printf("\nSuccessfully renamed %d files\n", renamedCount)
	}

	return nil
}

func generateUUIDName(originalName string) string {
	ext := filepath.Ext(originalName)
	newUUID := uuid.New().String()
	return newUUID + ext
}
