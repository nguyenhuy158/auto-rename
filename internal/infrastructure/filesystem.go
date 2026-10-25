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

func ListDirectory(basePath, currentPath string) ([]domain.FileEntry, error) {
	absBasePath, err := filepath.EvalSymlinks(basePath)
	if err != nil {
		return nil, err
	}
	absCurrentPath, err := ResolveWithin(absBasePath, currentPath)
	if err != nil {
		return nil, err
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

		relPath, err := filepath.Rel(absBasePath, absCurrentPath)
		if err != nil {
			continue
		}
		if relPath == "." {
			relPath = ""
		}
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

	sort.Slice(fileEntries, func(i, j int) bool {
		if fileEntries[i].IsDir != fileEntries[j].IsDir {
			return fileEntries[i].IsDir
		}
		return strings.ToLower(fileEntries[i].Name) < strings.ToLower(fileEntries[j].Name)
	})

	return fileEntries, nil
}

func ResolveWithin(basePath, relativePath string) (string, error) {
	base, err := filepath.EvalSymlinks(basePath)
	if err != nil {
		return "", err
	}
	target, err := filepath.EvalSymlinks(filepath.Join(base, relativePath))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(base, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", os.ErrPermission
	}
	return target, nil
}

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
