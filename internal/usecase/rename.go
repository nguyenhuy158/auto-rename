// Business logic for renaming files and cron scanning
package usecase

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"auto-rename/internal/config"
	"auto-rename/internal/domain"
	"auto-rename/internal/infrastructure"

	"github.com/google/uuid"
)

// renameFiles thực hiện đổi tên file trong thư mục
func RenameFiles(config config.Config, db *infrastructure.Database) error {
	log.Printf("Scanning directory: %s", config.Dir)
	if config.DryRun {
		log.Printf("DRY RUN MODE - No files will be renamed")
	}

	var files []os.DirEntry
	var err error
	if config.RenameSubfolder {
		err = filepath.WalkDir(config.Dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Chỉ xử lý file, không đổi tên folder
			if !d.IsDir() {
				files = append(files, &fileEntryWrapper{d, path})
			}
			return nil
		})
	} else {
		entries, e := os.ReadDir(config.Dir)
		if e != nil {
			err = e
		} else {
			for _, d := range entries {
				if !d.IsDir() {
					files = append(files, &fileEntryWrapper{d, filepath.Join(config.Dir, d.Name())})
				}
			}
		}
	}
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	log.Printf("Found %d files to process", len(files))
	log.Printf("Files: %v", files)
	renamedCount := 0
	skippedCount := 0
	for _, file := range files {
		// fileEntryWrapper đảm bảo có path đầy đủ
		name := file.Name()
		path := file.(interface{ Path() string }).Path()
		log.Printf("Scanning file: %s", path)
		if file.IsDir() {
			continue
		}

		if SameFileAsDB(config, file.Name()) {
			skippedCount++
			continue
		}

		if LooksLikeUUID(file.Name()) {
			skippedCount++
			continue
		}

		exists, err := db.HasOriginalName(file.Name())
		if err == nil && exists {
			skippedCount++
			continue
		}

		oldPath := path
		newName := GenerateUUIDName(name)
		newPath := filepath.Join(filepath.Dir(path), newName)

		fileSize, fileMode, modTime, err := infrastructure.GetFileInfo(oldPath)
		if err != nil {
			log.Printf("Failed to get file info for %s: %v", file.Name(), err)
			record := domain.FileRecord{
				OriginalName: file.Name(),
				NewName:      newName,
				FilePath:     config.Dir,
				Success:      false,
				ErrorMsg:     fmt.Sprintf("Failed to get file info: %v", err),
				RenamedAt:    time.Now().Format(time.RFC3339),
			}
			_ = db.InsertFileRecord(record)
			continue
		}

		log.Printf("  %s -> %s", file.Name(), newName)

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

		record := domain.FileRecord{
			OriginalName: file.Name(),
			NewName:      newName,
			FilePath:     config.Dir,
			FileSize:     fileSize,
			FileMode:     fileMode,
			ModTime:      modTime,
			Success:      success,
			ErrorMsg:     errorMsg,
			RenamedAt:    time.Now().Format(time.RFC3339),
		}

		_ = db.InsertFileRecord(record)

		if success {
			renamedCount++
		}
	}

	if config.DryRun {
		log.Printf("Would rename %d files (skipped %d)", renamedCount, skippedCount)
	} else {
		log.Printf("Successfully renamed %d files (skipped %d)", renamedCount, skippedCount)
	}

	return nil
}

// LooksLikeUUID kiểm tra tên file có phải dạng UUID
func LooksLikeUUID(name string) bool {
	ext := filepath.Ext(name)
	base := name[:len(name)-len(ext)]
	if len(base) != 36 {
		return false
	}
	expectedHyphens := []int{8, 13, 18, 23}
	for _, pos := range expectedHyphens {
		if pos >= len(base) || base[pos] != '-' {
			return false
		}
	}
	return true
}

// GenerateUUIDName tạo tên mới dạng UUID cho file
func GenerateUUIDName(originalName string) string {
	ext := filepath.Ext(originalName)
	newUUID := uuid.New().String()
	return newUUID + ext
}

// fileEntryWrapper dùng để lưu path đầy đủ cho file
type fileEntryWrapper struct {
	os.DirEntry
	fullPath string
}

func (f *fileEntryWrapper) Path() string {
	return f.fullPath
}

// SameFileAsDB kiểm tra file có phải là file database không
func SameFileAsDB(config config.Config, name string) bool {
	if config.DbPath == "" {
		return false
	}
	return filepath.Base(config.DbPath) == name
}

// startCronScanner launches a ticker that rescans the directory every minute.
func StartCronScanner(config config.Config, db *infrastructure.Database) {
	log.Printf("[cron] StartCronScanner initialized for dir=%s", config.Dir)
	ticker := time.NewTicker(time.Minute)
	for {
		log.Printf("[cron] waiting for next tick...")
		<-ticker.C
		log.Printf("[cron] running scan of %s", config.Dir)
		processed, skipped, err := RenameOnlyNewFiles(config, db)
		if err != nil {
			log.Printf("[cron] error: %v", err)
			log.Printf("[cron] run summary: processed=%d skipped=%d error=%v", processed, skipped, err)
		} else {
			log.Printf("[cron] scan complete")
			log.Printf("[cron] run summary: processed=%d skipped=%d error=nil", processed, skipped)
		}
	}
}

// RenameOnlyNewFiles chỉ đổi tên file chưa có trong DB
func RenameOnlyNewFiles(config config.Config, db *infrastructure.Database) (int, int, error) {
	var files []os.DirEntry
	var err error
	if config.RenameSubfolder {
		err = filepath.WalkDir(config.Dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !d.IsDir() {
				files = append(files, &fileEntryWrapper{d, path})
			}
			return nil
		})
		if err != nil {
			return 0, 0, fmt.Errorf("failed to scan directory: %w", err)
		}
	} else {
		entries, e := os.ReadDir(config.Dir)
		if e != nil {
			return 0, 0, fmt.Errorf("failed to read directory: %w", e)
		}
		for _, d := range entries {
			if !d.IsDir() {
				files = append(files, &fileEntryWrapper{d, filepath.Join(config.Dir, d.Name())})
			}
		}
	}

	processed := 0
	skipped := 0
	for _, file := range files {
		name := file.Name()
		path := file.(interface{ Path() string }).Path()
		if file.IsDir() {
			continue
		}
		if SameFileAsDB(config, name) {
			skipped++
			continue
		}
		if LooksLikeUUID(name) {
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
		oldPath := path
		newName := GenerateUUIDName(name)
		newPath := filepath.Join(filepath.Dir(path), newName)
		fileSize, fileMode, modTime, infoErr := infrastructure.GetFileInfo(oldPath)
		if infoErr != nil {
			log.Printf("[cron] info failed %s: %v", name, infoErr)
			record := domain.FileRecord{OriginalName: name, NewName: newName, FilePath: config.Dir, Success: false, ErrorMsg: fmt.Sprintf("file info: %v", infoErr), RenamedAt: time.Now().Format(time.RFC3339)}
			_ = db.InsertFileRecord(record)
			continue
		}
		if config.DryRun {
			log.Printf("[cron][dry-run] %s -> %s", name, newName)
		} else {
			if err := os.Rename(oldPath, newPath); err != nil {
				log.Printf("[cron] rename failed %s: %v", name, err)
				record := domain.FileRecord{OriginalName: name, NewName: newName, FilePath: config.Dir, FileSize: fileSize, FileMode: fileMode, ModTime: modTime, Success: false, ErrorMsg: err.Error(), RenamedAt: time.Now().Format(time.RFC3339)}
				_ = db.InsertFileRecord(record)
				continue
			}
			log.Printf("[cron] renamed %s -> %s", name, newName)
		}
		record := domain.FileRecord{OriginalName: name, NewName: newName, FilePath: config.Dir, FileSize: fileSize, FileMode: fileMode, ModTime: modTime, Success: true, RenamedAt: time.Now().Format(time.RFC3339)}
		if err := db.InsertFileRecord(record); err != nil {
			log.Printf("[cron] failed to record rename for %s: %v", name, err)
		}
		processed++
	}
	log.Printf("[cron] processed=%d skipped=%d", processed, skipped)
	return processed, skipped, nil
}
