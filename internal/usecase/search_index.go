package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"auto-rename/internal/infrastructure"
)

type IndexedFile struct {
	Path            string  `json:"path"`
	Name            string  `json:"name"`
	Kind            string  `json:"kind"`
	Size            int64   `json:"size"`
	ModifiedAt      string  `json:"modified_at"`
	DurationSeconds float64 `json:"duration_seconds"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	Codec           string  `json:"codec"`
	Format          string  `json:"format"`
	Snippet         string  `json:"snippet"`
	PDFText         string  `json:"-"`
}

type IndexStatus struct {
	Running        bool   `json:"running"`
	TotalFiles     int    `json:"total_files"`
	ProcessedFiles int    `json:"processed_files"`
	MediaIndexed   int    `json:"media_indexed"`
	PDFIndexed     int    `json:"pdf_indexed"`
	FailedFiles    int    `json:"failed_files"`
	UpdatedAt      string `json:"updated_at"`
}

type SearchIndex struct {
	root   string
	mutex  sync.RWMutex
	files  map[string]IndexedFile
	status IndexStatus
}

func NewSearchIndex(root string) *SearchIndex {
	return &SearchIndex{root: root, files: make(map[string]IndexedFile)}
}

func (s *SearchIndex) Trigger() bool {
	s.mutex.Lock()
	if s.status.Running {
		s.mutex.Unlock()
		return false
	}
	s.status.Running = true
	s.status.ProcessedFiles = 0
	s.status.MediaIndexed = 0
	s.status.PDFIndexed = 0
	s.status.FailedFiles = 0
	s.mutex.Unlock()
	go s.scan()
	return true
}

func (s *SearchIndex) Status() IndexStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.status
}

func (s *SearchIndex) Search(query string, limit int) []IndexedFile {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || limit <= 0 {
		return []IndexedFile{}
	}
	type ranked struct {
		file  IndexedFile
		score int
	}
	matches := make([]ranked, 0)
	s.mutex.RLock()
	for _, file := range s.files {
		name := strings.ToLower(file.Name)
		path := strings.ToLower(file.Path)
		metadata := strings.ToLower(file.Codec + " " + file.Format + " " + file.Kind + " " + strconv.Itoa(file.Width) + "x" + strconv.Itoa(file.Height) + " " + strconv.FormatFloat(file.DurationSeconds, 'f', 1, 64))
		text := strings.ToLower(file.PDFText)
		score := 0
		switch {
		case name == query:
			score = 5
		case strings.HasPrefix(name, query):
			score = 4
		case strings.Contains(name, query):
			score = 3
		case strings.Contains(path, query):
			score = 2
		case strings.Contains(metadata, query):
			score = 1
		case strings.Contains(text, query):
			score = 1
		}
		if score == 0 {
			continue
		}
		if at := strings.Index(text, query); at >= 0 {
			original := []rune(file.PDFText)
			position := utf8.RuneCountInString(text[:at])
			length := utf8.RuneCountInString(query)
			start := max(0, position-45)
			end := min(len(original), position+length+75)
			file.Snippet = strings.TrimSpace(string(original[start:end]))
		}
		file.PDFText = ""
		matches = append(matches, ranked{file: file, score: score})
	}
	s.mutex.RUnlock()
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].file.Path < matches[j].file.Path
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	result := make([]IndexedFile, 0, len(matches))
	for _, item := range matches {
		result = append(result, item.file)
	}
	return result
}

func (s *SearchIndex) scan() {
	root, err := filepath.EvalSymlinks(s.root)
	if err != nil {
		s.mutex.Lock()
		s.status.Running = false
		s.status.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		s.status.FailedFiles++
		s.mutex.Unlock()
		return
	}
	files := make([]IndexedFile, 0)
	filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			s.mutex.Lock()
			s.status.FailedFiles++
			s.mutex.Unlock()
			return nil
		}
		if entry == nil || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if path == root {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		image, video, _ := infrastructure.DetectMediaType(entry.Name())
		kind := "file"
		if entry.IsDir() {
			kind = "folder"
		} else if image {
			kind = "image"
		} else if video {
			kind = "video"
		} else if strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			kind = "pdf"
		}
		files = append(files, IndexedFile{Path: relative, Name: entry.Name(), Kind: kind, Size: info.Size(), ModifiedAt: info.ModTime().UTC().Format(time.RFC3339)})
		return nil
	})
	current := make(map[string]IndexedFile, len(files))
	for _, file := range files {
		current[file.Path] = file
	}
	s.mutex.Lock()
	s.files = current
	s.status.TotalFiles = len(files)
	s.mutex.Unlock()
	for _, file := range files {
		path, pathErr := infrastructure.ResolveWithin(root, file.Path)
		failed := false
		if pathErr != nil {
			failed = true
		} else if file.Kind == "video" || file.Kind == "image" {
			metadata, err := probeMedia(path)
			if err == nil {
				file.DurationSeconds = metadata.DurationSeconds
				file.Width = metadata.Width
				file.Height = metadata.Height
				file.Codec = metadata.Codec
				file.Format = metadata.Format
			} else {
				failed = true
			}
		}
		if file.Kind == "pdf" && !failed {
			content, err := extractPDF(path)
			if err == nil {
				file.PDFText = content
			} else {
				failed = true
			}
		}
		s.mutex.Lock()
		s.files[file.Path] = file
		s.status.ProcessedFiles++
		if (file.Kind == "video" || file.Kind == "image") && !failed {
			s.status.MediaIndexed++
		}
		if file.Kind == "pdf" && !failed {
			s.status.PDFIndexed++
		}
		if failed {
			s.status.FailedFiles++
		}
		s.mutex.Unlock()
	}
	s.mutex.Lock()
	s.status.Running = false
	s.status.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	s.mutex.Unlock()
}

type mediaProbe struct {
	DurationSeconds float64
	Width           int
	Height          int
	Codec           string
	Format          string
}

func probeMedia(path string) (mediaProbe, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration,format_name:stream=codec_name,codec_type,width,height", "-of", "json", path).Output()
	if err != nil || len(output) > 64<<10 {
		return mediaProbe{}, errors.New("media metadata unavailable")
	}
	var raw struct {
		Format struct {
			Duration string `json:"duration"`
			Name     string `json:"format_name"`
		} `json:"format"`
		Streams []struct {
			Codec  string `json:"codec_name"`
			Kind   string `json:"codec_type"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &raw); err != nil {
		return mediaProbe{}, err
	}
	duration, _ := strconv.ParseFloat(raw.Format.Duration, 64)
	result := mediaProbe{DurationSeconds: duration, Format: raw.Format.Name}
	for _, stream := range raw.Streams {
		if stream.Kind == "video" {
			result.Codec, result.Width, result.Height = stream.Codec, stream.Width, stream.Height
			break
		}
	}
	return result, nil
}

type boundedOutput struct {
	bytes.Buffer
	limit int
}

func (b *boundedOutput) Write(data []byte) (int, error) {
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		return 0, errors.New("output limit exceeded")
	}
	if len(data) > remaining {
		written, _ := b.Buffer.Write(data[:remaining])
		return written, errors.New("output limit exceeded")
	}
	return b.Buffer.Write(data)
}

func extractPDF(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	output := &boundedOutput{limit: 256 << 10}
	command := exec.CommandContext(ctx, "pdftotext", "-layout", "-nopgbrk", path, "-")
	command.Stdout = output
	command.Stderr = &bytes.Buffer{}
	err := command.Run()
	if err != nil && output.Len() == 0 {
		return "", err
	}
	return output.String(), nil
}
