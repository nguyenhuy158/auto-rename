package usecase

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"auto-rename/internal/config"
	"auto-rename/internal/domain"
	"auto-rename/internal/infrastructure"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type Scanner struct {
	config    config.Config
	db        *infrastructure.Database
	root      string
	mutex     sync.Mutex
	current   domain.ScanRun
	running   bool
	blocked   bool
	scheduler *cron.Cron
	groups    []DuplicateGroup
}

func NewScanner(cfg config.Config, db *infrastructure.Database) (*Scanner, error) {
	root, err := filepath.EvalSymlinks(cfg.Dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, errors.New("scan directory is unavailable")
	}
	quarantine := filepath.Clean(cfg.QuarantineDir)
	if !filepath.IsAbs(quarantine) {
		return nil, errors.New("quarantine directory must be absolute")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(quarantine))
	if err != nil {
		return nil, err
	}
	quarantine = filepath.Join(parent, filepath.Base(quarantine))
	relative, err := filepath.Rel(root, quarantine)
	if err != nil || (relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))) {
		return nil, errors.New("quarantine directory must be outside scan directory")
	}
	return &Scanner{config: cfg, db: db, root: root}, nil
}

func (s *Scanner) StartCron() error {
	if !s.config.Cron || s.config.WebOnly {
		return nil
	}
	if s.blocked {
		return nil
	}
	scheduler := cron.New()
	if _, err := scheduler.AddFunc(s.config.CronExpr, func() { _, _ = s.Trigger("cron") }); err != nil {
		return err
	}
	scheduler.Start()
	s.scheduler = scheduler
	return nil
}

func (s *Scanner) Trigger(source string) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.running {
		return 0, errors.New("a scan is already running")
	}
	if s.blocked {
		return 0, errors.New("unresolved file operation needs review")
	}
	id, err := s.db.StartRun(source)
	if err != nil {
		return 0, err
	}
	s.current = domain.ScanRun{ID: id, Source: source, Status: "running", StartedAt: time.Now().UTC().Format(time.RFC3339)}
	s.running = true
	go s.execute(id)
	return id, nil
}

func (s *Scanner) Status() domain.ScanRun {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.current
}

func (s *Scanner) Configuration() config.Config {
	return s.config
}

func (s *Scanner) setRun(run domain.ScanRun) {
	s.mutex.Lock()
	s.current = run
	s.mutex.Unlock()
}

func (s *Scanner) eligible(path string, entry os.DirEntry) bool {
	if !entry.Type().IsRegular() {
		return false
	}
	if LooksLikeUUID(entry.Name()) {
		return false
	}
	extension := strings.ToLower(filepath.Ext(path))
	for _, allowed := range s.config.Extensions {
		if extension == strings.ToLower(strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

func (s *Scanner) paths() ([]string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(s.root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != s.root && !s.config.RenameSubfolder {
				return filepath.SkipDir
			}
			return nil
		}
		if s.eligible(path, entry) {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

func (s *Scanner) execute(id int) {
	run := s.Status()
	paths, err := s.paths()
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		s.finish(run)
		return
	}
	run.TotalFiles = len(paths)
	restored, err := s.db.RestoredNames()
	if err != nil {
		run.Status = "failed"
		run.Error = "D1 unavailable"
		run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		s.finish(run)
		return
	}
	_ = s.db.UpdateRun(run)
	s.setRun(run)
	for _, path := range paths {
		renamed, err := s.rename(path, restored)
		if err != nil {
			run.Error = err.Error()
			if !s.config.DryRun {
				run.Status = "stopped_on_error"
				run.ProcessedFiles++
				break
			}
		} else if renamed {
			run.RenamedFiles++
		}
		run.ProcessedFiles++
		s.setRun(run)
		if run.ProcessedFiles%10 == 0 {
			_ = s.db.UpdateRun(run)
		}
	}
	if run.Status == "running" {
		run.Status = "completed"
	}
	if run.Error != "" && run.Status == "completed" {
		run.Status = "completed_with_errors"
	}
	if s.config.DryRun && run.Error == "" {
		run.Status = "preview"
	}
	run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	s.finish(run)
}

func (s *Scanner) finish(run domain.ScanRun) {
	_ = s.db.UpdateRun(run)
	s.mutex.Lock()
	s.current = run
	s.running = false
	s.mutex.Unlock()
}

func (s *Scanner) rename(path string, restored map[string]bool) (bool, error) {
	if err := s.checkPath(path); err != nil {
		return false, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, errors.New("file changed during scan")
	}
	if restored[path] {
		return false, nil
	}
	if s.config.DryRun {
		return false, nil
	}
	newName := uuid.NewString() + filepath.Ext(path)
	newPath := filepath.Join(filepath.Dir(path), newName)
	record := domain.FileRecord{
		OriginalName: filepath.Base(path), NewName: newName, FilePath: filepath.Dir(path),
		FileSize: info.Size(), FileMode: info.Mode().String(), ModTime: info.ModTime().UTC().Format(time.RFC3339),
		RenamedAt: time.Now().UTC().Format(time.RFC3339), Status: "planned",
	}
	id, err := s.db.CreateFileRecord(record)
	if err != nil {
		return false, err
	}
	if err := moveNoReplace(path, newPath); err != nil {
		_ = s.db.UpdateFileRecord(id, "failed", false, err.Error(), "")
		return false, err
	}
	return true, s.db.UpdateFileRecord(id, "renamed", true, "", "")
}

func (s *Scanner) RestoreName(id int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.running {
		return errors.New("scan is running")
	}
	if s.blocked {
		return errors.New("unresolved file operation needs review")
	}
	record, err := s.db.GetFileRecord(id)
	if err != nil {
		return err
	}
	if record.Status != "renamed" {
		return errors.New("record is not eligible for rollback")
	}
	source := filepath.Join(record.FilePath, record.NewName)
	target := filepath.Join(record.FilePath, record.OriginalName)
	if err := s.checkPath(source); err != nil {
		return err
	}
	if err := s.checkPath(target); err != nil {
		return err
	}
	if err := s.db.UpdateFileRecord(id, "restore_pending", true, "", ""); err != nil {
		return err
	}
	if err := moveNoReplace(source, target); err != nil {
		_ = s.db.UpdateFileRecord(id, "renamed", true, err.Error(), "")
		return err
	}
	return s.db.UpdateFileRecord(id, "restored", true, "", time.Now().UTC().Format(time.RFC3339))
}

func (s *Scanner) checkPath(path string) error {
	clean := filepath.Clean(path)
	relative, err := filepath.Rel(s.root, clean)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return errors.New("path is outside scan directory")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(clean))
	if err != nil {
		return err
	}
	relative, err = filepath.Rel(s.root, parent)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return errors.New("path resolves outside scan directory")
	}
	return nil
}

func moveNoReplace(source, target string) error {
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("target already exists: %s", target)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Link(source, target); err != nil {
		return err
	}
	if err := os.Remove(source); err != nil {
		return fmt.Errorf("source removal failed after linking target: %w", err)
	}
	return nil
}

func LooksLikeUUID(name string) bool {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	_, err := uuid.Parse(base)
	return err == nil
}
