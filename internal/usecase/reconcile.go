package usecase

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"auto-rename/internal/domain"
)

func filePresent(path string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, errors.New("path is not a regular file")
	}
	return true, nil
}

func (s *Scanner) Reconcile() error {
	records, err := s.db.PendingFileRecords()
	if err != nil {
		return err
	}
	actions, err := s.db.PendingQuarantines()
	if err != nil {
		return err
	}
	unresolved := 0
	for _, record := range records {
		if record.Status == "needs_review" {
			unresolved++
			continue
		}
		original := filepath.Join(record.FilePath, record.OriginalName)
		renamed := filepath.Join(record.FilePath, record.NewName)
		originalExists, originalErr := filePresent(original)
		renamedExists, renamedErr := filePresent(renamed)
		if originalErr != nil || renamedErr != nil || originalExists == renamedExists {
			unresolved++
			_ = s.db.UpdateFileRecord(record.Id, "needs_review", record.Success, "manual review required", record.RestoredAt)
			continue
		}
		if record.Status == "planned" {
			if renamedExists {
				err = s.db.UpdateFileRecord(record.Id, "renamed", true, "", "")
			} else {
				err = s.db.UpdateFileRecord(record.Id, "failed", false, "operation was not applied", "")
			}
		} else if originalExists {
			err = s.db.UpdateFileRecord(record.Id, "restored", true, "", time.Now().UTC().Format(time.RFC3339))
		} else {
			err = s.db.UpdateFileRecord(record.Id, "renamed", true, "", "")
		}
		if err != nil {
			return err
		}
	}
	for _, action := range actions {
		if action.Status == "needs_review" {
			unresolved++
			continue
		}
		originalExists, originalErr := filePresent(action.OriginalPath)
		quarantineExists, quarantineErr := filePresent(action.QuarantinePath)
		if originalErr != nil || quarantineErr != nil || originalExists == quarantineExists {
			unresolved++
			_ = s.db.UpdateQuarantine(action.ID, "needs_review")
			continue
		}
		if action.Status == "planned" {
			if quarantineExists {
				err = s.db.UpdateQuarantine(action.ID, "quarantined")
			} else {
				err = s.db.UpdateQuarantine(action.ID, "failed")
			}
		} else if originalExists {
			err = s.db.RestoreQuarantine(action.ID)
		} else {
			err = s.db.UpdateQuarantine(action.ID, "quarantined")
		}
		if err != nil {
			return err
		}
	}
	if unresolved > 0 {
		s.mutex.Lock()
		s.blocked = true
		s.current = domain.ScanRun{Status: "needs_review", Error: "unresolved file operations require manual review"}
		s.mutex.Unlock()
	}
	return nil
}
