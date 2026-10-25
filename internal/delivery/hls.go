package delivery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"auto-rename/internal/infrastructure"
)

type hlsSession struct {
	path     string
	dir      string
	cancel   context.CancelFunc
	done     chan struct{}
	lastUsed time.Time
}

type hlsService struct {
	root     string
	mutex    sync.Mutex
	sessions map[string]*hlsSession
}

func newHLSService(root string) *hlsService {
	service := &hlsService{root: root, sessions: make(map[string]*hlsSession)}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			service.mutex.Lock()
			service.cleanExpired()
			service.mutex.Unlock()
		}
	}()
	return service
}

func (s *hlsService) cleanExpired() {
	now := time.Now()
	for id, session := range s.sessions {
		if now.Sub(session.lastUsed) > 3*time.Minute {
			session.cancel()
			<-session.done
			os.RemoveAll(session.dir)
			delete(s.sessions, id)
		}
	}
}

func (s *hlsService) start(path string) (string, error) {
	abs, err := infrastructure.ResolveWithin(s.root, path)
	if err != nil {
		return "", err
	}
	_, video, _ := infrastructure.DetectMediaType(abs)
	if !video {
		return "", errors.New("video required")
	}
	info, err := os.Stat(abs)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("video unavailable")
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.cleanExpired()
	for id, session := range s.sessions {
		if session.path == abs {
			session.lastUsed = time.Now()
			return id, nil
		}
	}
	if len(s.sessions) >= 2 {
		return "", errors.New("stream capacity reached")
	}
	dir, err := os.MkdirTemp("", "auto-rename-hls-")
	if err != nil {
		return "", err
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	id := hex.EncodeToString(token[:])
	ctx, cancel := context.WithCancel(context.Background())
	session := &hlsSession{path: abs, dir: dir, cancel: cancel, done: make(chan struct{}), lastUsed: time.Now()}
	command := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-nostdin", "-re", "-i", abs, "-map", "0:v:0", "-map", "0:a:0?", "-vf", "scale=w='min(1280,iw)':h='min(720,ih)':force_original_aspect_ratio=decrease:force_divisible_by=2", "-c:v", "libx264", "-preset", "ultrafast", "-crf", "28", "-threads", "1", "-force_key_frames", "expr:gte(t,n_forced*3)", "-c:a", "aac", "-b:a", "96k", "-f", "hls", "-hls_time", "3", "-hls_list_size", "6", "-hls_delete_threshold", "2", "-hls_flags", "delete_segments", "-hls_segment_filename", filepath.Join(dir, "segment%05d.ts"), filepath.Join(dir, "index.m3u8"))
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		cancel()
		os.RemoveAll(dir)
		return "", err
	}
	s.sessions[id] = session
	go func() { command.Wait(); close(session.done) }()
	return id, nil
}

func (s *hlsService) get(id string) *hlsSession {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	session := s.sessions[id]
	if session != nil {
		session.lastUsed = time.Now()
	}
	return session
}

func (ws *WebServer) ensureHLS() *hlsService {
	ws.hlsOnce.Do(func() { ws.hls = newHLSService(ws.baseDir) })
	return ws.hls
}

func (ws *WebServer) handleHLSStart(w http.ResponseWriter, r *http.Request) {
	if !requirePOST(w, r) {
		return
	}
	if ws.baseDir == "" {
		http.Error(w, "File browser not configured", http.StatusBadRequest)
		return
	}
	var input struct {
		Path string `json:"path"`
	}
	if err := readJSON(r, &input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	id, err := ws.ensureHLS().start(input.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, map[string]string{"url": "/api/hls?id=" + id + "&file=index.m3u8"})
}

func (ws *WebServer) handleHLS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, name := r.URL.Query().Get("id"), r.URL.Query().Get("file")
	if len(id) != 32 || (name != "index.m3u8" && !validSegment(name)) {
		http.NotFound(w, r)
		return
	}
	session := ws.ensureHLS().get(id)
	if session == nil {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(session.dir, name)
	if name == "index.m3u8" {
		deadline := time.Now().Add(15 * time.Second)
		for {
			if _, err := os.Stat(path); err == nil {
				break
			}
			select {
			case <-session.done:
				http.Error(w, "Stream failed", http.StatusUnprocessableEntity)
				return
			case <-r.Context().Done():
				return
			default:
			}
			if time.Now().After(deadline) {
				http.Error(w, "Stream timeout", http.StatusGatewayTimeout)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, "Stream unavailable", http.StatusServiceUnavailable)
			return
		}
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if validSegment(line) {
				lines[i] = fmt.Sprintf("/api/hls?id=%s&file=%s", id, line)
			}
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store")
		io.WriteString(w, strings.Join(lines, "\n"))
		return
	}
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "private, max-age=60")
	http.ServeFile(w, r, path)
}

func validSegment(name string) bool {
	if !strings.HasPrefix(name, "segment") || !strings.HasSuffix(name, ".ts") || len(name) != len("segment00000.ts") {
		return false
	}
	for _, digit := range name[len("segment") : len(name)-len(".ts")] {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}
