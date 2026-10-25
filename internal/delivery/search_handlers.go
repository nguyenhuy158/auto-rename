package delivery

import (
	"auto-rename/internal/domain"
	"auto-rename/internal/infrastructure"
	"auto-rename/internal/usecase"
	"net/http"
	"strings"
)

type searchNavigation struct {
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	URL      string `json:"url"`
}

func (ws *WebServer) ensureIndex() *usecase.SearchIndex {
	ws.indexOnce.Do(func() {
		if ws.baseDir != "" {
			ws.index = usecase.NewSearchIndex(ws.baseDir)
			ws.index.Trigger()
		}
	})
	return ws.index
}

func (ws *WebServer) handleIndexStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	index := ws.ensureIndex()
	if index == nil {
		http.Error(w, "File browser not configured", http.StatusBadRequest)
		return
	}
	writeJSON(w, index.Status())
}

func (ws *WebServer) handleIndexTrigger(w http.ResponseWriter, r *http.Request) {
	if !requirePOST(w, r) {
		return
	}
	index := ws.ensureIndex()
	if index == nil {
		http.Error(w, "File browser not configured", http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"started": index.Trigger()})
}

func (ws *WebServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > 128 {
		http.Error(w, "Query too long", http.StatusBadRequest)
		return
	}
	index := ws.ensureIndex()
	if index == nil {
		http.Error(w, "File browser not configured", http.StatusBadRequest)
		return
	}
	files := index.Search(query, 12)
	databaseResults := infrastructure.SearchResults{Records: []domain.FileRecord{}, Runs: []domain.ScanRun{}, Actions: []domain.QuarantineAction{}}
	if query != "" {
		var err error
		databaseResults, err = ws.db.SearchAll(query)
		if err != nil {
			http.Error(w, "Search unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	links := make([]searchNavigation, 0)
	if query != "" {
		settings := ws.scanner.Configuration()
		items := []searchNavigation{
			{Kind: "page", Title: "Tổng quan", Subtitle: "Dashboard và tiến trình", URL: "/"},
			{Kind: "page", Title: "File trùng", Subtitle: "Nhóm trùng nội dung", URL: "/duplicates"},
			{Kind: "page", Title: "Lịch sử đổi tên", Subtitle: "Bản ghi và hoàn tác", URL: "/records"},
			{Kind: "page", Title: "Nhật ký chạy", Subtitle: "Các lần quét", URL: "/logs"},
			{Kind: "page", Title: "Duyệt thư mục", Subtitle: "File và ảnh/video", URL: "/browse"},
			{Kind: "page", Title: "Cấu hình", Subtitle: "settings.json · Cài đặt đang dùng", URL: "/settings"},
			{Kind: "setting", Title: "Thư mục nguồn", Subtitle: settings.Dir, URL: "/settings"},
			{Kind: "setting", Title: "Thư mục cách ly", Subtitle: settings.QuarantineDir, URL: "/settings"},
			{Kind: "setting", Title: "Định dạng video", Subtitle: strings.Join(settings.Extensions, ", "), URL: "/settings"},
			{Kind: "setting", Title: "Xem trước", Subtitle: boolLabel(settings.DryRun), URL: "/settings"},
			{Kind: "setting", Title: "Lịch tự động", Subtitle: boolLabel(settings.Cron) + " " + settings.CronExpr, URL: "/settings"},
		}
		lower := strings.ToLower(query)
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.Title+" "+item.Subtitle), lower) {
				links = append(links, item)
			}
		}
	}
	groups := make([]map[string]any, 0)
	if query != "" {
		for _, group := range ws.scanner.DuplicateGroups() {
			for _, file := range group.Files {
				if strings.Contains(strings.ToLower(file.Path), strings.ToLower(query)) {
					groups = append(groups, map[string]any{"sha256": group.SHA256, "path": file.Path, "files": len(group.Files)})
					break
				}
			}
			if len(groups) >= 5 {
				break
			}
		}
	}
	writeJSON(w, map[string]any{"files": files, "navigation": links, "duplicates": groups, "records": nonNil(databaseResults.Records), "runs": nonNil(databaseResults.Runs), "actions": nonNil(databaseResults.Actions), "index": index.Status()})
}

func boolLabel(value bool) string {
	if value {
		return "Bật"
	}
	return "Tắt"
}
