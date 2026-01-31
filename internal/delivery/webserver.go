// Web server delivery layer for auto-rename
package delivery

import (
	"auto-rename/internal/domain"
	"auto-rename/internal/infrastructure"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"
)

type WebServer struct {
	db      *infrastructure.Database
	webPort string
}

func NewWebServer(db *infrastructure.Database, webPort string) *WebServer {
	return &WebServer{db: db, webPort: webPort}
}

func (ws *WebServer) Start() error {
	mux := http.NewServeMux()
	// Serve static files from ./static directory
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	mux.HandleFunc("/favicon.ico", ws.handleFavicon)
	mux.HandleFunc("/logs", ws.handleLogs)
	mux.HandleFunc("/records", ws.handleRecord)
	mux.HandleFunc("/api/records", ws.handleAPIRecords)
	mux.HandleFunc("/api/stats", ws.handleAPIStats)
	mux.HandleFunc("/api/cron/status", ws.handleAPICronStatus)
	mux.HandleFunc("/", ws.handleIndex)
	server := &http.Server{
		Addr:    ":" + ws.webPort,
		Handler: mux,
	}
	return server.ListenAndServe()
}

func (ws *WebServer) handleFavicon(w http.ResponseWriter, r *http.Request) {
	// Try to serve actual favicon if present
	if _, err := os.Stat("static/favicon.ico"); err == nil {
		http.ServeFile(w, r, "static/favicon.ico")
		return
	}
	// Fallback: minimal empty icon bytes so browsers stop requesting
	w.Header().Set("Content-Type", "image/x-icon")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10,
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x18, 0x00,
		0x28, 0x01, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00,
	})
}

func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	println("handleIndex:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))

	// Fetch all statistics for SSR
	total, _ := ws.db.GetTotalRecords()
	successful, _ := ws.db.GetSuccessfulRenames()
	failed, _ := ws.db.GetFailedRenames()
	recent, _ := ws.db.GetRecentActivity(24)
	recentOps, _ := ws.db.GetRecentOperations(10)
	cronStatus := infrastructure.GetCronStatus()

	// Prepare dashboard data
	data := &domain.DashboardData{
		TotalRecords:      total,
		SuccessfulRenames: successful,
		FailedRenames:     failed,
		RecentActivity:    recent,
		RecentOperations:  recentOps,
		CronStatus:        cronStatus,
		CurrentTime:       time.Now().Format("2006-01-02 15:04:05"),
	}

	tmpl, err := template.ParseFiles("template/index.html")
	if err != nil {
		http.Error(w, "Template index error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func (ws *WebServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	println("handleLogs:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))

	// Parse query parameters
	page := 1
	pageSize := 50
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	// Fetch filtered and paginated logs
	records, _ := ws.db.GetRecordsByFilter(page, pageSize, search, status, "", "")
	total, _ := ws.db.GetRecordCountByFilter(search, status, "", "")

	// Prepare logs page data
	data := &domain.LogsPageData{
		Records:      records,
		TotalRecords: total,
		CurrentPage:  page,
		PageSize:     pageSize,
		SearchQuery:  search,
		StatusFilter: status,
	}

	tmpl, err := template.ParseFiles("template/logs.html")
	if err != nil {
		http.Error(w, "Template logs error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func (ws *WebServer) handleRecord(w http.ResponseWriter, r *http.Request) {
	println("handleRecord:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))

	// Parse query parameters
	page := 1
	pageSize := 20
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	dateFrom := r.URL.Query().Get("dateFrom")
	dateTo := r.URL.Query().Get("dateTo")

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	// Fetch filtered and paginated records
	records, _ := ws.db.GetRecordsByFilter(page, pageSize, search, status, dateFrom, dateTo)
	total, _ := ws.db.GetRecordCountByFilter(search, status, dateFrom, dateTo)

	// Prepare records page data
	data := &domain.RecordsPageData{
		Records:      records,
		TotalRecords: total,
		CurrentPage:  page,
		PageSize:     pageSize,
		SearchQuery:  search,
		StatusFilter: status,
		DateFrom:     dateFrom,
		DateTo:       dateTo,
	}

	tmpl, err := template.ParseFiles("template/records.html")
	if err != nil {
		http.Error(w, "Template records error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func (ws *WebServer) handleAPIRecords(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	page := 1
	pageSize := 20
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	dateFrom := r.URL.Query().Get("dateFrom")
	dateTo := r.URL.Query().Get("dateTo")

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}

	records, err := ws.db.GetRecordsByFilter(page, pageSize, search, status, dateFrom, dateTo)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	total, err := ws.db.GetRecordCountByFilter(search, status, dateFrom, dateTo)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"records":  records,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (ws *WebServer) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	total, _ := ws.db.GetTotalRecords()
	successful, _ := ws.db.GetSuccessfulRenames()
	failed, _ := ws.db.GetFailedRenames()
	recent, _ := ws.db.GetRecentActivity(24)
	recentOps, _ := ws.db.GetRecentOperations(10)

	stats := map[string]interface{}{
		"status":              "ok",
		"total_records":       total,
		"successful_renames":  successful,
		"failed_renames":      failed,
		"recent_activity":     recent,
		"recent_operations":   recentOps,
		"timestamp":           time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(stats)
}

func (ws *WebServer) handleAPICronStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	cronStatus := infrastructure.GetCronStatus()
	json.NewEncoder(w).Encode(cronStatus)
}
