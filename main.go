package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Config struct {
	Dir     string
	DryRun  bool
	WebPort string
	WebOnly bool
	DbPath  string
	Cron    bool // If true, run rename scan every minute continuously
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
	if !config.WebOnly && config.Dir != "" {
		if err := renameFiles(config, db); err != nil {
			log.Fatalf("Error renaming files: %v", err)
		}
	}

	// If cron mode enabled, start periodic scan goroutine
	if config.Cron {
		if config.Dir == "" {
			log.Fatalf("-cron requires -dir to be specified")
		}
		log.Printf("Cron mode enabled: scanning %s every 60s", config.Dir)
		InitializeCronStatus(true, config.Dir)
		go startCronScanner(config, db)
	}

	// Start web server in background if port is specified
	if config.WebPort != "" {
		fmt.Printf("\nStarting web server on port %s...\n", config.WebPort)
		fmt.Printf("View results at: http://localhost:%s\n", config.WebPort)

		webServer := NewWebServer(db, config.WebPort)
		log.Fatal(webServer.Start())
	}
}

// startCronScanner launches a ticker that rescans the directory every minute.
// It only attempts to rename files that have not yet been processed.
func startCronScanner(config Config, db *Database) {
	ticker := time.NewTicker(time.Minute)
	for {
		<-ticker.C
		MarkCronStart()
		log.Printf("[cron] running scan of %s", config.Dir)
		// Run a specialized scan that skips already processed files
		processed, skipped, err := renameOnlyNewFiles(config, db)
		if err != nil {
			log.Printf("[cron] error: %v", err)
			MarkCronComplete(processed, skipped, err)
			log.Printf("[cron] run summary: processed=%d skipped=%d error=%v", processed, skipped, err)
		} else {
			log.Printf("[cron] scan complete")
			MarkCronComplete(processed, skipped, nil)
			log.Printf("[cron] run summary: processed=%d skipped=%d error=nil", processed, skipped)
		}
	}
}

// renameOnlyNewFiles scans the directory and only renames files that are not in the DB yet.
func renameOnlyNewFiles(config Config, db *Database) (int, int, error) {
	files, err := os.ReadDir(config.Dir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read directory: %w", err)
	}

	processed := 0
	skipped := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if sameFileAsDB(config, name) { // never rename the database itself
			skipped++
			continue
		}
		// Skip if looks like already a UUID (simple heuristic: 36 chars with hyphens)
		if looksLikeUUID(name) {
			skipped++
			continue
		}
		exists, err := db.HasOriginalName(name)
		if err != nil {
			log.Printf("[cron] db lookup failed for %s: %v", name, err)
			continue
		}
		if exists {
			skipped++
			continue
		}
		// Reuse existing rename logic for single file by temporarily constructing slice? For simplicity call internal logic
		// We'll replicate minimal part of renameFiles for this one file.
		oldPath := filepath.Join(config.Dir, name)
		newName := generateUUIDName(name)
		newPath := filepath.Join(config.Dir, newName)
		fileSize, fileMode, modTime, infoErr := getFileInfo(oldPath)
		if infoErr != nil {
			log.Printf("[cron] info failed %s: %v", name, infoErr)
			record := FileRecord{OriginalName: name, NewName: newName, FilePath: config.Dir, Success: false, ErrorMsg: fmt.Sprintf("file info: %v", infoErr), RenamedAt: time.Now()}
			_ = db.InsertFileRecord(record)
			continue
		}
		if config.DryRun {
			log.Printf("[cron][dry-run] %s -> %s", name, newName)
		} else {
			if err := os.Rename(oldPath, newPath); err != nil {
				log.Printf("[cron] rename failed %s: %v", name, err)
				record := FileRecord{OriginalName: name, NewName: newName, FilePath: config.Dir, FileSize: fileSize, FileMode: fileMode, ModTime: modTime, Success: false, ErrorMsg: err.Error(), RenamedAt: time.Now()}
				_ = db.InsertFileRecord(record)
				continue
			}
			log.Printf("[cron] renamed %s -> %s", name, newName)
		}
		record := FileRecord{OriginalName: name, NewName: newName, FilePath: config.Dir, FileSize: fileSize, FileMode: fileMode, ModTime: modTime, Success: true, RenamedAt: time.Now()}
		if err := db.InsertFileRecord(record); err != nil {
			log.Printf("[cron] failed to record rename for %s: %v", name, err)
		}
		processed++
	}
	log.Printf("[cron] processed=%d skipped=%d", processed, skipped)
	return processed, skipped, nil
}

// looksLikeUUID does a basic length and hyphen position check (not a full validation) and also preserves extension awareness.
func looksLikeUUID(name string) bool {
	ext := filepath.Ext(name)
	base := name[:len(name)-len(ext)]
	if len(base) != 36 { // 8-4-4-4-12 pattern total length 36
		return false
	}
	// quick hyphen position check
	expectedHyphens := []int{8, 13, 18, 23}
	for _, pos := range expectedHyphens {
		if pos >= len(base) || base[pos] != '-' {
			return false
		}
	}
	return true
}

func parseFlags() Config {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	// Get environment variables with defaults
	envDir := os.Getenv("DIR")
	envDryRun := getBoolEnv("DRY_RUN", false)
	envWebPort := getEnv("WEB_PORT", "8080")
	envWebOnly := getBoolEnv("WEB_ONLY", false)
	envDbPath := getEnv("DB_PATH", "./file_renames.db")
	envCron := getBoolEnv("CRON", false)

	var config Config
	flag.StringVar(&config.Dir, "dir", envDir, "Directory containing files to rename (can also set DIR env var)")
	flag.BoolVar(&config.DryRun, "dry-run", envDryRun, "Preview changes without actually renaming files (can also set DRY_RUN env var)")
	flag.StringVar(&config.WebPort, "web-port", envWebPort, "Port for web interface (can also set WEB_PORT env var)")
	flag.BoolVar(&config.WebOnly, "web-only", envWebOnly, "Only start web server without renaming files (can also set WEB_ONLY env var)")
	flag.StringVar(&config.DbPath, "db", envDbPath, "SQLite database path (can also set DB_PATH env var)")
	flag.BoolVar(&config.Cron, "cron", envCron, "Continuously scan directory every minute (can also set CRON env var)")
	flag.Parse()

	return config
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getBoolEnv retrieves a boolean environment variable or returns a default value
func getBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
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
	skippedCount := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if sameFileAsDB(config, file.Name()) {
			skippedCount++
			continue
		}

		// Skip if already UUID-like (assume already processed)
		if looksLikeUUID(file.Name()) {
			skippedCount++
			continue
		}

		// Skip if database already has a record for this original name
		exists, err := db.HasOriginalName(file.Name())
		if err == nil && exists {
			skippedCount++
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
		fmt.Printf("\nWould rename %d files (skipped %d)\n", renamedCount, skippedCount)
	} else {
		fmt.Printf("\nSuccessfully renamed %d files (skipped %d)\n", renamedCount, skippedCount)
	}

	return nil
}

func generateUUIDName(originalName string) string {
	ext := filepath.Ext(originalName)
	newUUID := uuid.New().String()
	return newUUID + ext
}

// sameFileAsDB checks if the given filename refers to the sqlite database file (basename compare).
func sameFileAsDB(config Config, name string) bool {
	if config.DbPath == "" {
		return false
	}
	return filepath.Base(config.DbPath) == name
}
