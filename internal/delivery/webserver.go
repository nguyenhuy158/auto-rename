package delivery

import (
	"auto-rename/internal/domain"
	"auto-rename/internal/infrastructure"
	"auto-rename/internal/usecase"
	"encoding/json"
	"html/template"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

type WebServer struct {
	db      *infrastructure.Database
	webPort string
	baseDir string
	scanner *usecase.Scanner
}

func NewWebServer(db *infrastructure.Database, webPort string) *WebServer {
	return &WebServer{db: db, webPort: webPort, baseDir: ""}
}

func NewWebServerWithDir(db *infrastructure.Database, webPort, baseDir string) *WebServer {
	return &WebServer{db: db, webPort: webPort, baseDir: baseDir}
}

func NewWebServerWithScanner(db *infrastructure.Database, scanner *usecase.Scanner, webPort, baseDir string) *WebServer {
	return &WebServer{db: db, scanner: scanner, webPort: webPort, baseDir: baseDir}
}

func (ws *WebServer) Start() error {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	mux.Handle("/assets/", http.FileServer(http.Dir("web/dist")))
	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "openapi.yaml") })
	mux.HandleFunc("/favicon.ico", ws.handleFavicon)
	mux.HandleFunc("/logs", ws.handleApp)
	mux.HandleFunc("/records", ws.handleApp)
	mux.HandleFunc("/browse", ws.handleApp)
	mux.HandleFunc("/api/records", ws.handleAPIRecords)
	mux.HandleFunc("/api/browse", ws.handleAPIBrowser)
	mux.HandleFunc("/api/stats", ws.handleAPIStats)
	mux.HandleFunc("/api/health", ws.handleHealth)
	mux.HandleFunc("/api/cron/status", ws.handleAPICronStatus)
	mux.HandleFunc("/api/cron/trigger", ws.handleAPICronTrigger)
	mux.HandleFunc("/api/runs", ws.handleRuns)
	mux.HandleFunc("/api/duplicates/scan", ws.handleDuplicateScan)
	mux.HandleFunc("/api/duplicates/quarantine", ws.handleDuplicateQuarantine)
	mux.HandleFunc("/api/duplicates/quarantine-all", ws.handleDuplicateQuarantineAll)
	mux.HandleFunc("/api/duplicates", ws.handleDuplicates)
	mux.HandleFunc("/api/quarantine/restore", ws.handleQuarantineRestore)
	mux.HandleFunc("/api/quarantine", ws.handleQuarantine)
	mux.HandleFunc("/api/records/restore", ws.handleRecordRestore)
	mux.HandleFunc("/duplicates", ws.handleApp)
	mux.HandleFunc("/api/file", ws.handleAPIFile)
	mux.HandleFunc("/settings", ws.handleApp)
	mux.HandleFunc("/api/settings", ws.handleAPISettings)
	mux.HandleFunc("/", ws.handleApp)
	server := &http.Server{
		Addr:    ":" + ws.webPort,
		Handler: mux,
	}
	return server.ListenAndServe()
}

func (ws *WebServer) handleApp(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/duplicates" && r.URL.Path != "/records" && r.URL.Path != "/logs" && r.URL.Path != "/browse" && r.URL.Path != "/settings" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "web/dist/index.html")
}

func (ws *WebServer) handleFavicon(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat("static/favicon.ico"); err == nil {
		http.ServeFile(w, r, "static/favicon.ico")
		return
	}
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

	total, _ := ws.db.GetTotalRecords()
	successful, _ := ws.db.GetSuccessfulRenames()
	failed, _ := ws.db.GetFailedRenames()
	recent, _ := ws.db.GetRecentActivity(24)
	recentOps, _ := ws.db.GetRecentOperations(10)
	run := ws.scanner.Status()
	configuration := ws.scanner.Configuration()
	cronStatus := &domain.CronStatus{
		Enabled:        configuration.Cron && !configuration.WebOnly,
		IsRunning:      run.Status == "running",
		Directory:      ws.baseDir,
		LastRun:        run.StartedAt,
		FilesProcessed: run.ProcessedFiles,
		FilesSkipped:   run.TotalFiles - run.RenamedFiles,
		LastError:      run.Error,
	}

	data := &domain.DashboardData{
		TotalRecords:      total,
		SuccessfulRenames: successful,
		FailedRenames:     failed,
		RecentActivity:    recent,
		RecentOperations:  recentOps,
		CronStatus:        cronStatus,
		CurrentTime:       time.Now().Format("2006-01-02 15:04:05"),
	}

	funcs := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}

	tmpl, err := template.New("index.html").Funcs(funcs).ParseFiles("template/index.html")
	if err != nil {
		http.Error(w, "Template index error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func (ws *WebServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	println("handleLogs:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))

	page := 1
	pageSize := 50
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	records, _ := ws.db.GetRecordsByFilter(page, pageSize, search, status, "", "")
	total, _ := ws.db.GetRecordCountByFilter(search, status, "", "")

	data := &domain.LogsPageData{
		Records:      records,
		TotalRecords: total,
		CurrentPage:  page,
		PageSize:     pageSize,
		SearchQuery:  search,
		StatusFilter: status,
	}

	funcs := template.FuncMap{
		"ceil": func(a float64) int { return int(math.Ceil(a)) },
		"div":  func(a, b int) float64 { return float64(a) / float64(b) },
		"mul":  func(a, b int) int { return a * b },
		"add":  func(a, b int) int { return a + b },
		"sub":  func(a, b int) int { return a - b },
		"eq":   func(a, b string) bool { return a == b },
		"lt":   func(a, b int) bool { return a < b },
		"gt":   func(a, b int) bool { return a > b },
	}

	tmpl, err := template.New("logs.html").Funcs(funcs).ParseFiles("template/logs.html")
	if err != nil {
		http.Error(w, "Template logs error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func (ws *WebServer) handleRecord(w http.ResponseWriter, r *http.Request) {
	println("handleRecord:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))

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

	records, _ := ws.db.GetRecordsByFilter(page, pageSize, search, status, dateFrom, dateTo)
	total, _ := ws.db.GetRecordCountByFilter(search, status, dateFrom, dateTo)

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

	funcs := template.FuncMap{
		"ceil": func(a float64) int { return int(math.Ceil(a)) },
		"div":  func(a, b int) float64 { return float64(a) / float64(b) },
		"mul":  func(a, b int) int { return a * b },
		"add":  func(a, b int) int { return a + b },
		"sub":  func(a, b int) int { return a - b },
		"eq":   func(a, b string) bool { return a == b },
		"lt":   func(a, b int) bool { return a < b },
		"gt":   func(a, b int) bool { return a > b },
	}

	tmpl, err := template.New("records.html").Funcs(funcs).ParseFiles("template/records.html")
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
		"status":             "ok",
		"total_records":      total,
		"successful_renames": successful,
		"failed_renames":     failed,
		"recent_activity":    recent,
		"recent_operations":  recentOps,
		"timestamp":          time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(stats)
}

func (ws *WebServer) handleAPICronStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ws.scanner.Status())
}

func (ws *WebServer) handleAPICronTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if ws.scanner == nil {
		http.Error(w, "Scanner unavailable", http.StatusServiceUnavailable)
		return
	}
	id, err := ws.scanner.Trigger("manual")
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "running", "run_id": id})
}

func (ws *WebServer) handleAPIFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	if ws.baseDir == "" {
		http.Error(w, "File browser not configured", http.StatusBadRequest)
		return
	}

	absFilePath, err := infrastructure.ResolveWithin(ws.baseDir, filePath)
	if err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	info, err := os.Stat(absFilePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if info.IsDir() {
		http.Error(w, "Path is a directory", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, absFilePath)
}

func (ws *WebServer) handleBrowser(w http.ResponseWriter, r *http.Request) {
	println("handleBrowser:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))

	currentPath := r.URL.Query().Get("path")
	if currentPath == "" {
		currentPath = "."
	}

	if ws.baseDir == "" {
		http.Error(w, "File browser not configured (no directory specified)", http.StatusBadRequest)
		return
	}

	entries, err := infrastructure.ListDirectory(ws.baseDir, currentPath)
	if err != nil {
		http.Error(w, "Failed to list directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	breadcrumb := infrastructure.GetBreadcrumb(currentPath)

	data := &domain.FileBrowserData{
		CurrentPath: currentPath,
		Entries:     entries,
		BaseDir:     ws.baseDir,
		Breadcrumb:  breadcrumb,
	}

	funcs := template.FuncMap{
		"div": func(a, b int64) float64 { return float64(a) / float64(b) },
	}

	tmpl, err := template.New("browse.html").Funcs(funcs).ParseFiles("template/browse.html")
	if err != nil {
		http.Error(w, "Template browse error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func (ws *WebServer) handleAPIBrowser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if ws.baseDir == "" {
		http.Error(w, "File browser not configured", http.StatusBadRequest)
		return
	}

	currentPath := r.URL.Query().Get("path")
	if currentPath == "" {
		currentPath = "."
	}

	entries, err := infrastructure.ListDirectory(ws.baseDir, currentPath)
	if err != nil {
		http.Error(w, "Failed to list directory", http.StatusInternalServerError)
		return
	}

	breadcrumb := infrastructure.GetBreadcrumb(currentPath)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"current_path": currentPath,
		"entries":      entries,
		"breadcrumb":   breadcrumb,
		"base_dir":     ws.baseDir,
	})
}
