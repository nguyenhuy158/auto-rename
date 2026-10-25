package delivery

import (
	"auto-rename/internal/infrastructure"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const thumbnailBudget = 32 << 20

type thumbnailService struct {
	mutex sync.Mutex
	items map[string][]byte
	order []string
	bytes int
	limit chan struct{}
}

func newThumbnailService() *thumbnailService {
	return &thumbnailService{items: make(map[string][]byte), limit: make(chan struct{}, 2)}
}

func (s *thumbnailService) get(key string) []byte {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.items[key]
}

func (s *thumbnailService) put(key string, data []byte) {
	if len(data) > thumbnailBudget {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, exists := s.items[key]; exists {
		return
	}
	for s.bytes+len(data) > thumbnailBudget && len(s.order) > 0 {
		oldest := s.order[0]
		s.order = s.order[1:]
		s.bytes -= len(s.items[oldest])
		delete(s.items, oldest)
	}
	s.items[key] = data
	s.order = append(s.order, key)
	s.bytes += len(data)
}

func (ws *WebServer) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if ws.baseDir == "" || r.URL.Query().Get("path") == "" {
		http.Error(w, "File path required", http.StatusBadRequest)
		return
	}
	path, err := infrastructure.ResolveWithin(ws.baseDir, r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		http.Error(w, "File unavailable", http.StatusNotFound)
		return
	}
	image, video, _ := infrastructure.DetectMediaType(path)
	ext := strings.ToLower(filepath.Ext(path))
	if (!image && !video) || ext == ".svg" || ext == ".ico" {
		http.Error(w, "Thumbnail unavailable", http.StatusUnsupportedMediaType)
		return
	}
	ws.thumbnailOnce.Do(func() { ws.thumbnail = newThumbnailService() })
	key := path + ":" + info.ModTime().UTC().Format(time.RFC3339Nano) + ":" + stringSize(info.Size())
	hash := sha256.Sum256([]byte(key))
	etag := `"` + hex.EncodeToString(hash[:]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, no-cache")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	data := ws.thumbnail.get(key)
	if data == nil {
		select {
		case ws.thumbnail.limit <- struct{}{}:
			defer func() { <-ws.thumbnail.limit }()
		case <-r.Context().Done():
			return
		}
		data = ws.thumbnail.get(key)
		if data == nil {
			data, err = renderThumbnail(r.Context(), path, video)
			if err != nil {
				http.Error(w, "Thumbnail unavailable", http.StatusUnprocessableEntity)
				return
			}
			ws.thumbnail.put(key, data)
		}
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(data)
}

func stringSize(value int64) string {
	return strconv.FormatInt(value, 10)
}

func renderThumbnail(parent context.Context, path string, video bool) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	input := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-threads", "1"}
	if video {
		input = append(input, "-ss", "0.5")
	}
	input = append(input, "-i", path, "-frames:v", "1", "-vf", "scale=320:320:force_original_aspect_ratio=decrease", "-q:v", "6", "-f", "image2pipe", "-vcodec", "mjpeg", "pipe:1")
	output, err := exec.CommandContext(ctx, "ffmpeg", input...).Output()
	if video && (err != nil || len(output) == 0) && ctx.Err() == nil {
		input = []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-threads", "1", "-i", path, "-frames:v", "1", "-vf", "scale=320:320:force_original_aspect_ratio=decrease", "-q:v", "6", "-f", "image2pipe", "-vcodec", "mjpeg", "pipe:1"}
		output, err = exec.CommandContext(ctx, "ffmpeg", input...).Output()
	}
	if err != nil || len(output) == 0 || len(output) > 1<<20 || !bytes.HasPrefix(output, []byte{0xff, 0xd8}) {
		return nil, errors.New("thumbnail conversion failed")
	}
	return output, nil
}
