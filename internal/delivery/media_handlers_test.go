package delivery

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestThumbnailReturnsBoundedJPEGAndRejectsOutsidePath(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	root := t.TempDir()
	imagePath := filepath.Join(root, "preview.png")
	file, err := os.Create(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	preview := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			preview.Set(x, y, color.RGBA{R: 120, G: 70, B: 210, A: 255})
		}
	}
	if err := png.Encode(file, preview); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	ws := NewWebServerWithDir(nil, "", root)
	request := httptest.NewRequest(http.MethodGet, "/api/thumbnail?path=preview.png", nil)
	response := httptest.NewRecorder()
	ws.handleThumbnail(response, request)
	if response.Code != http.StatusOK || response.Body.Len() == 0 || response.Body.Len() > 1<<20 || !bytes.HasPrefix(response.Body.Bytes(), []byte{0xff, 0xd8}) {
		t.Fatalf("thumbnail response: status=%d size=%d", response.Code, response.Body.Len())
	}
	cached := httptest.NewRecorder()
	request.Header.Set("If-None-Match", response.Header().Get("ETag"))
	ws.handleThumbnail(cached, request)
	if cached.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", cached.Code)
	}
	outside := filepath.Join(t.TempDir(), "outside.png")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.png")); err != nil {
		t.Fatal(err)
	}
	denied := httptest.NewRecorder()
	ws.handleThumbnail(denied, httptest.NewRequest(http.MethodGet, "/api/thumbnail?path=escape.png", nil))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", denied.Code)
	}
}

func TestBrowsePaginatesAndFiltersInsideSource(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"alpha.mp4", "beta.mov", "gamma.pdf"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ws := NewWebServerWithDir(nil, "", root)
	query := url.Values{"path": {"."}, "search": {"a"}, "page": {"2"}, "pageSize": {"1"}, "sort": {"name"}, "order": {"asc"}}
	response := httptest.NewRecorder()
	ws.handleAPIBrowser(response, httptest.NewRequest(http.MethodGet, "/api/browse?"+query.Encode(), nil))
	if response.Code != http.StatusOK {
		t.Fatalf("browse response: %d", response.Code)
	}
	var result struct {
		Entries []struct {
			Name string `json:"name"`
		} `json:"entries"`
		Total int `json:"total"`
		Page  int `json:"page"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 3 || result.Page != 2 || len(result.Entries) != 1 || result.Entries[0].Name != "beta.mov" {
		t.Fatalf("unexpected browse page: total=%d page=%d entries=%v", result.Total, result.Page, result.Entries)
	}
}

func TestHLSStreamsVideoWithoutWritingSource(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	root := t.TempDir()
	video := filepath.Join(root, "clip.mkv")
	if output, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=64x64:r=10", "-t", "4", "-c:v", "mpeg4", video).CombinedOutput(); err != nil {
		t.Fatalf("create fixture: %v %s", err, output)
	}
	before, err := os.ReadFile(video)
	if err != nil {
		t.Fatal(err)
	}
	ws := NewWebServerWithDir(nil, "", root)
	start := httptest.NewRecorder()
	ws.handleHLSStart(start, httptest.NewRequest(http.MethodPost, "/api/hls/start", strings.NewReader(`{"path":"clip.mkv"}`)))
	if start.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", start.Code, start.Body.String())
	}
	var result struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	playlist := httptest.NewRecorder()
	ws.handleHLS(playlist, httptest.NewRequest(http.MethodGet, result.URL, nil))
	if playlist.Code != http.StatusOK || !strings.Contains(playlist.Body.String(), "/api/hls?id=") {
		t.Fatalf("playlist status=%d body=%s", playlist.Code, playlist.Body.String())
	}
	var segmentURL string
	for _, line := range strings.Split(playlist.Body.String(), "\n") {
		if strings.Contains(line, "segment") {
			segmentURL = line
			break
		}
	}
	if segmentURL == "" {
		t.Fatal("playlist has no segment")
	}
	segment := httptest.NewRecorder()
	ws.handleHLS(segment, httptest.NewRequest(http.MethodGet, segmentURL, nil))
	if segment.Code != http.StatusOK || segment.Body.Len() == 0 || segment.Header().Get("Content-Type") != "video/mp2t" {
		t.Fatalf("segment response: status=%d size=%d", segment.Code, segment.Body.Len())
	}
	after, err := os.ReadFile(video)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("source video changed")
	}
	for _, session := range ws.hls.sessions {
		session.cancel()
		<-session.done
		os.RemoveAll(session.dir)
	}
	denied := httptest.NewRecorder()
	ws.handleHLSStart(denied, httptest.NewRequest(http.MethodPost, "/api/hls/start", strings.NewReader(`{"path":"../clip.mkv"}`)))
	if denied.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected denied path, got %d", denied.Code)
	}
}
