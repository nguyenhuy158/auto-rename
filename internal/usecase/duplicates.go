package usecase

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"auto-rename/internal/domain"

	"github.com/google/uuid"
)

type DuplicateFile struct {
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
}

type DuplicateGroup struct {
	SHA256 string          `json:"sha256"`
	Size   int64           `json:"size"`
	Files  []DuplicateFile `json:"files"`
}

type quarantineSelection struct {
	hash   string
	keeper string
	path   string
}

func (s *Scanner) DuplicateGroups() []DuplicateGroup {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return append([]DuplicateGroup(nil), s.groups...)
}

func (s *Scanner) TriggerDuplicateScan() (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.running {
		return 0, errors.New("another operation is running")
	}
	if s.blocked {
		return 0, errors.New("unresolved file operation needs review")
	}
	id, err := s.db.StartRun("duplicate_scan")
	if err != nil {
		return 0, err
	}
	s.current = domain.ScanRun{ID: id, Source: "duplicate_scan", Status: "running", StartedAt: time.Now().UTC().Format(time.RFC3339)}
	s.running = true
	go s.analyzeDuplicates()
	return id, nil
}

func (s *Scanner) videoPaths() ([]string, error) {
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
		if !entry.Type().IsRegular() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		for _, allowed := range s.config.Extensions {
			if extension == strings.ToLower(strings.TrimSpace(allowed)) {
				paths = append(paths, path)
				break
			}
		}
		return nil
	})
	return paths, err
}

func hashFile(path string) (string, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !before.Mode().IsRegular() {
		return "", errors.New("file is not regular")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	after, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return "", errors.New("file changed while hashing")
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (s *Scanner) analyzeDuplicates() {
	run := s.Status()
	paths, err := s.videoPaths()
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		s.finish(run)
		return
	}
	run.TotalFiles = len(paths)
	_ = s.db.UpdateRun(run)
	sizes := make(map[int64][]DuplicateFile)
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			run.ProcessedFiles++
			continue
		}
		sizes[info.Size()] = append(sizes[info.Size()], DuplicateFile{Path: path, Size: info.Size(), ModifiedAt: info.ModTime().UTC().Format(time.RFC3339Nano)})
	}
	hashes := make(map[string][]DuplicateFile)
	for _, files := range sizes {
		if len(files) == 1 {
			run.ProcessedFiles++
			continue
		}
		for _, file := range files {
			hash, err := hashFile(file.Path)
			if err != nil {
				run.Error = err.Error()
			} else {
				hashes[hash] = append(hashes[hash], file)
			}
			run.ProcessedFiles++
			s.setRun(run)
			if run.ProcessedFiles%10 == 0 {
				_ = s.db.UpdateRun(run)
			}
		}
	}
	groups := make([]DuplicateGroup, 0)
	for hash, files := range hashes {
		if len(files) < 2 {
			continue
		}
		sort.Slice(files, func(i, j int) bool {
			if files[i].ModifiedAt == files[j].ModifiedAt {
				return files[i].Path < files[j].Path
			}
			return files[i].ModifiedAt < files[j].ModifiedAt
		})
		groups = append(groups, DuplicateGroup{SHA256: hash, Size: files[0].Size, Files: files})
		run.DuplicateFiles += len(files) - 1
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].SHA256 < groups[j].SHA256 })
	s.mutex.Lock()
	s.groups = groups
	s.mutex.Unlock()
	run.Status = "completed"
	if run.Error != "" {
		run.Status = "completed_with_errors"
	}
	run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	s.finish(run)
}

func (s *Scanner) QuarantineDuplicates(hash string, selected []string, all bool) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.running {
		return 0, errors.New("another operation is running")
	}
	if s.blocked {
		return 0, errors.New("unresolved file operation needs review")
	}
	var group *DuplicateGroup
	for i := range s.groups {
		if s.groups[i].SHA256 == hash {
			group = &s.groups[i]
			break
		}
	}
	if group == nil {
		return 0, errors.New("duplicate group not found; scan again")
	}
	keeper := group.Files[0].Path
	allowed := make(map[string]bool)
	for _, file := range group.Files[1:] {
		allowed[file.Path] = true
	}
	if all {
		selected = selected[:0]
		for _, file := range group.Files[1:] {
			selected = append(selected, file.Path)
		}
	}
	if len(selected) == 0 {
		return 0, errors.New("no duplicate files selected")
	}
	seen := make(map[string]bool)
	for _, path := range selected {
		if !allowed[path] || seen[path] {
			return 0, errors.New("selection must exclude the oldest file")
		}
		seen[path] = true
	}
	selections := make([]quarantineSelection, 0, len(selected))
	for _, path := range selected {
		selections = append(selections, quarantineSelection{hash: hash, keeper: keeper, path: path})
	}
	return s.startQuarantine(selections)
}

func (s *Scanner) QuarantineAllDuplicates() (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.running {
		return 0, errors.New("another operation is running")
	}
	if s.blocked {
		return 0, errors.New("unresolved file operation needs review")
	}
	selections := make([]quarantineSelection, 0)
	for _, group := range s.groups {
		for _, file := range group.Files[1:] {
			selections = append(selections, quarantineSelection{hash: group.SHA256, keeper: group.Files[0].Path, path: file.Path})
		}
	}
	if len(selections) == 0 {
		return 0, errors.New("no duplicate files found; scan again")
	}
	return s.startQuarantine(selections)
}

func (s *Scanner) startQuarantine(selections []quarantineSelection) (int, error) {
	id, err := s.db.StartRun("quarantine")
	if err != nil {
		return 0, err
	}
	s.current = domain.ScanRun{ID: id, Source: "quarantine", Status: "running", StartedAt: time.Now().UTC().Format(time.RFC3339), TotalFiles: len(selections)}
	s.running = true
	go s.quarantineFiles(selections)
	return id, nil
}

func (s *Scanner) quarantineFiles(selections []quarantineSelection) {
	run := s.Status()
	if err := os.MkdirAll(s.config.QuarantineDir, 0700); err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		s.finish(run)
		return
	}
	realQuarantine, err := filepath.EvalSymlinks(s.config.QuarantineDir)
	if err != nil {
		run.Status = "failed"
		run.Error = "quarantine directory cannot be resolved"
		run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		s.finish(run)
		return
	}
	relative, err := filepath.Rel(s.root, realQuarantine)
	if err != nil || (relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))) {
		run.Status = "failed"
		run.Error = "quarantine directory resolves inside scan directory"
		run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		s.finish(run)
		return
	}
	rootInfo, rootErr := os.Stat(s.root)
	quarantineInfo, quarantineErr := os.Stat(realQuarantine)
	if rootErr != nil || quarantineErr != nil || rootInfo.Sys().(*syscall.Stat_t).Dev != quarantineInfo.Sys().(*syscall.Stat_t).Dev {
		run.Status = "failed"
		run.Error = "quarantine directory must be on the same filesystem"
		run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		s.finish(run)
		return
	}
	for _, selection := range selections {
		if err := s.quarantineOne(selection.hash, selection.keeper, selection.path); err != nil {
			run.Error = err.Error()
			run.Status = "stopped_on_error"
			run.ProcessedFiles++
			break
		}
		run.ProcessedFiles++
		s.setRun(run)
		_ = s.db.UpdateRun(run)
	}
	s.mutex.Lock()
	s.groups = nil
	s.mutex.Unlock()
	if run.Status == "running" {
		run.Status = "completed"
	}
	run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	s.finish(run)
}

func (s *Scanner) quarantineOne(hash, keeper, path string) error {
	keeperHash, err := hashFile(keeper)
	if err != nil || keeperHash != hash {
		return errors.New("keeper changed; scan duplicates again")
	}
	if err := s.checkPath(path); err != nil {
		return err
	}
	currentHash, err := hashFile(path)
	if err != nil {
		return err
	}
	if currentHash != hash {
		return errors.New("duplicate changed; scan again")
	}
	target := filepath.Join(s.config.QuarantineDir, uuid.NewString()+filepath.Ext(path))
	action := domain.QuarantineAction{OriginalPath: path, QuarantinePath: target, SHA256: hash, KeptPath: keeper, Status: "planned", CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	id, err := s.db.CreateQuarantine(action)
	if err != nil {
		return err
	}
	if err := moveNoReplace(path, target); err != nil {
		return err
	}
	return s.db.UpdateQuarantine(id, "quarantined")
}

func (s *Scanner) RestoreQuarantine(id int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.running {
		return errors.New("another operation is running")
	}
	if s.blocked {
		return errors.New("unresolved file operation needs review")
	}
	action, err := s.db.GetQuarantine(id)
	if err != nil {
		return err
	}
	if action.Status != "quarantined" {
		return errors.New("file is not quarantined")
	}
	if err := s.checkPath(action.OriginalPath); err != nil {
		return err
	}
	relative, err := filepath.Rel(filepath.Clean(s.config.QuarantineDir), filepath.Clean(action.QuarantinePath))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return errors.New("invalid quarantine path")
	}
	if err := s.db.UpdateQuarantine(id, "restore_pending"); err != nil {
		return err
	}
	if err := moveNoReplace(action.QuarantinePath, action.OriginalPath); err != nil {
		_ = s.db.UpdateQuarantine(id, "quarantined")
		return err
	}
	return s.db.RestoreQuarantine(id)
}
