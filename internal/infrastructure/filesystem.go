// File system helpers for auto-rename
package infrastructure

import (
	"auto-rename/internal/domain"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	imageExtensions = map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".png": "image/png", ".gif": "image/gif",
		".webp": "image/webp", ".svg": "image/svg+xml",
		".ico": "image/x-icon", ".bmp": "image/bmp",
	}
	videoExtensions = map[string]string{
		".mp4": "video/mp4", ".webm": "video/webm",
		".mov": "video/quicktime", ".avi": "video/x-msvideo",
		".mkv": "video/x-matroska", ".flv": "video/x-flv",
		".wmv": "video/x-ms-wmv", ".m4v": "video/x-m4v",
	}
)

func GetFileInfo(path string) (int64, string, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, "", "", err
	}
	return info.Size(), info.Mode().String(), info.ModTime().Format("2006-01-02 15:04:05"), nil
}

// DetectMediaType determines if a file is an image or video
func DetectMediaType(filename string) (isImage, isVideo bool, mimeType string) {
	ext := strings.ToLower(filepath.Ext(filename))
	
	if mime, ok := imageExtensions[ext]; ok {
		return true, false, mime
	}
	if mime, ok := videoExtensions[ext]; ok {
		return false, true, mime
	}
	return false, false, ""
}

// ListDirectory lists files and directories in a given path
func ListDirectory(basePath, currentPath string) ([]domain.FileEntry, error) {
	// Prevent path traversal attacks
	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}

	absCurrentPath := filepath.Join(absBasePath, currentPath)
	absCurrentPath, err = filepath.Abs(absCurrentPath)
	if err != nil {
		return nil, err
	}

	// Ensure we're still within the base directory
	if !strings.HasPrefix(absCurrentPath, absBasePath) {
		return nil, os.ErrPermission
	}

	entries, err := os.ReadDir(absCurrentPath)
	if err != nil {
		return nil, err
	}

	var fileEntries []domain.FileEntry

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Relative path from base directory
		relPath := strings.TrimPrefix(absCurrentPath, absBasePath)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		if relPath != "" {
			relPath = filepath.Join(relPath, entry.Name())
		} else {
			relPath = entry.Name()
		}

		isImage, isVideo, mimeType := false, false, ""
		if !entry.IsDir() {
			isImage, isVideo, mimeType = DetectMediaType(entry.Name())
		}

		fileEntries = append(fileEntries, domain.FileEntry{
			Name:      entry.Name(),
			Path:      relPath,
			IsDir:     entry.IsDir(),
			Size:      info.Size(),
			ModTime:   info.ModTime().Format("2006-01-02 15:04:05"),
			CanRename: !entry.IsDir(),
			IsImage:   isImage,
			IsVideo:   isVideo,
			MimeType:  mimeType,
		})
	}

	// Sort: directories first, then alphabetically
	sort.Slice(fileEntries, func(i, j int) bool {
		if fileEntries[i].IsDir != fileEntries[j].IsDir {
			return fileEntries[i].IsDir
		}
		return strings.ToLower(fileEntries[i].Name) < strings.ToLower(fileEntries[j].Name)
	})

	return fileEntries, nil
}

// GetBreadcrumb generates breadcrumb items for navigation
func GetBreadcrumb(currentPath string) []domain.BreadcrumbItem {
	var breadcrumb []domain.BreadcrumbItem

	breadcrumb = append(breadcrumb, domain.BreadcrumbItem{
		Name: "Root",
		Path: "",
	})

	if currentPath == "" || currentPath == "." {
		return breadcrumb
	}

	parts := strings.Split(strings.TrimPrefix(currentPath, string(filepath.Separator)), string(filepath.Separator))
	for i, part := range parts {
		if part == "" {
			continue
		}
		path := filepath.Join(parts[:i+1]...)
		breadcrumb = append(breadcrumb, domain.BreadcrumbItem{
			Name: part,
			Path: path,
		})
	}

	return breadcrumb
}
