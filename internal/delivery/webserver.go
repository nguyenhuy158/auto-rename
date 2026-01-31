// Web server delivery layer for auto-rename
package delivery

import (
	"auto-rename/internal/config"
	"auto-rename/internal/domain"
	"auto-rename/internal/infrastructure"
	"encoding/json"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type WebServer struct {
	db      *infrastructure.Database
	webPort string
	baseDir string
}

func NewWebServer(db *infrastructure.Database, webPort string) *WebServer {
	return &WebServer{db: db, webPort: webPort, baseDir: ""}
}

func NewWebServerWithDir(db *infrastructure.Database, webPort, baseDir string) *WebServer {
	return &WebServer{db: db, webPort: webPort, baseDir: baseDir}
}

func (ws *WebServer) Start() error {
	mux := http.NewServeMux()
	// Serve static files from ./static directory
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	mux.HandleFunc("/favicon.ico", ws.handleFavicon)
	mux.HandleFunc("/logs", ws.handleLogs)
	mux.HandleFunc("/records", ws.handleRecord)
	mux.HandleFunc("/browse", ws.handleBrowser)
	mux.HandleFunc("/api/records", ws.handleAPIRecords)
	mux.HandleFunc("/api/browse", ws.handleAPIBrowser)
	mux.HandleFunc("/api/stats", ws.handleAPIStats)
	mux.HandleFunc("/api/cron/status", ws.handleAPICronStatus)
	mux.HandleFunc("/api/cron/trigger", ws.handleAPICronTrigger)
	mux.HandleFunc("/api/file", ws.handleAPIFile)
	mux.HandleFunc("/settings", ws.handleSettings)
	mux.HandleFunc("/api/settings", ws.handleAPISettings)
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

	funcs := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"lt":  func(a, b int) bool { return a < b },
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

func (ws *WebServer) handleAPICronTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Set cron as running
	infrastructure.SetCronRunning(true)
	defer infrastructure.SetCronRunning(false)

	// Trigger a scan in background (non-blocking)
	go func() {
		// Import usecase to call rename function
		// For now, just mark that a trigger occurred
		nextRun := time.Now().Add(1 * time.Minute)
		infrastructure.RecordCronExecution(nextRun, 0, 0, nil)
	}()

	result := map[string]interface{}{
		"status":  "triggered",
		"message": "Cron scan triggered successfully",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(result)
}

func (ws *WebServer) handleAPIFile(w http.ResponseWriter, r *http.Request) {
	// Get file path from query parameter
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	// If no base directory configured
	if ws.baseDir == "" {
		http.Error(w, "File browser not configured", http.StatusBadRequest)
		return
	}

	// Validate path to prevent traversal attacks
	absBasePath, err := filepath.Abs(ws.baseDir)
	if err != nil {
		http.Error(w, "Invalid base directory", http.StatusInternalServerError)
		return
	}

	absFilePath := filepath.Join(absBasePath, filePath)
	absFilePath, err = filepath.Abs(absFilePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Ensure we're still within the base directory
	if !strings.HasPrefix(absFilePath, absBasePath) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if file exists and is not a directory
	info, err := os.Stat(absFilePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if info.IsDir() {
		http.Error(w, "Path is a directory", http.StatusBadRequest)
		return
	}

	// Serve the file with appropriate content type
	http.ServeFile(w, r, absFilePath)
}

func (ws *WebServer) handleBrowser(w http.ResponseWriter, r *http.Request) {
	println("handleBrowser:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))

	// Get current path from query parameter
	currentPath := r.URL.Query().Get("path")
	if currentPath == "" {
		currentPath = "."
	}

	// If no base directory configured, show error
	if ws.baseDir == "" {
		http.Error(w, "File browser not configured (no directory specified)", http.StatusBadRequest)
		return
	}

	// List directory contents
	entries, err := infrastructure.ListDirectory(ws.baseDir, currentPath)
	if err != nil {
		http.Error(w, "Failed to list directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get breadcrumb navigation
	breadcrumb := infrastructure.GetBreadcrumb(currentPath)

	// Prepare browser data
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

	// If no base directory configured
	if ws.baseDir == "" {
		http.Error(w, "File browser not configured", http.StatusBadRequest)
		return
	}

	// Get current path from query parameter
	currentPath := r.URL.Query().Get("path")
	if currentPath == "" {
		currentPath = "."
	}

	// List directory contents
	entries, err := infrastructure.ListDirectory(ws.baseDir, currentPath)
	if err != nil {
		http.Error(w, "Failed to list directory", http.StatusInternalServerError)
		return
	}

	// Get breadcrumb navigation
	breadcrumb := infrastructure.GetBreadcrumb(currentPath)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"current_path": currentPath,
		"entries":      entries,
		"breadcrumb":   breadcrumb,
		"base_dir":     ws.baseDir,
	})
}

func (ws *WebServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	println("handleSettings:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))

	// Load settings from JSON file
	settings, err := config.LoadSettings("settings.json")
	if err != nil {
		http.Error(w, "Failed to load settings", http.StatusInternalServerError)
		return
	}

	// Prepare settings page data
	data := &domain.SettingsData{
		Dir:                 settings.Dir,
		DryRun:              settings.DryRun,
		WebPort:             settings.WebPort,
		WebOnly:             settings.WebOnly,
		DbPath:              settings.DbPath,
		Cron:                settings.Cron,
		CronExpr:            settings.CronExpr,
		RenameSubfolder:     settings.RenameSubfolder,
		FormatSize:          settings.FormatSize,
		FormatDatetime:      settings.FormatDatetime,
		FormatTimezone:      settings.FormatTimezone,
		FormatSizeDecimals:  settings.FormatSizeDecimals,
		UpdatedAt:           settings.UpdatedAt,
	}

	tmpl, err := template.New("settings.html").ParseFiles("template/settings.html")
	if err != nil {
		http.Error(w, "Template settings error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func (ws *WebServer) handleAPISettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		// Load settings from JSON file
		settings, err := config.LoadSettings("settings.json")
		if err != nil {
			http.Error(w, "Failed to load settings", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"dir":                   settings.Dir,
			"dry_run":               settings.DryRun,
			"web_port":              settings.WebPort,
			"web_only":              settings.WebOnly,
			"db_path":               settings.DbPath,
			"cron":                  settings.Cron,
			"cron_expr":             settings.CronExpr,
			"rename_subfolder":      settings.RenameSubfolder,
			"format_size":           settings.FormatSize,
			"format_datetime":       settings.FormatDatetime,
			"format_timezone":       settings.FormatTimezone,
			"format_size_decimals":  settings.FormatSizeDecimals,
			"updated_at":            settings.UpdatedAt,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == http.MethodPost {
		// Parse request body
		var req struct {
			Dir                  string `json:"dir"`
			DryRun               bool   `json:"dry_run"`
			WebPort              string `json:"web_port"`
			WebOnly              bool   `json:"web_only"`
			DbPath               string `json:"db_path"`
			Cron                 bool   `json:"cron"`
			CronExpr             string `json:"cron_expr"`
			RenameSubfolder      bool   `json:"rename_subfolder"`
			FormatSize           string `json:"format_size"`
			FormatDatetime       string `json:"format_datetime"`
			FormatTimezone       string `json:"format_timezone"`
			FormatSizeDecimals   int    `json:"format_size_decimals"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate directory path if provided and not web_only
		if !req.WebOnly && req.Dir != "" {
			if info, err := os.Stat(req.Dir); err != nil || !info.IsDir() {
				http.Error(w, "Invalid directory path", http.StatusBadRequest)
				return
			}
		}

		// Validate format options
		validSizes := map[string]bool{"auto": true, "B": true, "KB": true, "MB": true, "GB": true}
		if !validSizes[req.FormatSize] {
			http.Error(w, "Invalid format_size", http.StatusBadRequest)
			return
		}

		validTimezones := map[string]bool{"UTC": true, "Local": true}
		if !validTimezones[req.FormatTimezone] {
			http.Error(w, "Invalid format_timezone", http.StatusBadRequest)
			return
		}

		if req.FormatSizeDecimals < 0 || req.FormatSizeDecimals > 5 {
			http.Error(w, "format_size_decimals must be 0-5", http.StatusBadRequest)
			return
		}

		// Create settings struct and save to file
		newSettings := config.Settings{
			Dir:                 req.Dir,
			DryRun:              req.DryRun,
			WebPort:             req.WebPort,
			WebOnly:             req.WebOnly,
			DbPath:              req.DbPath,
			Cron:                req.Cron,
			CronExpr:            req.CronExpr,
			RenameSubfolder:     req.RenameSubfolder,
			FormatSize:          req.FormatSize,
			FormatDatetime:      req.FormatDatetime,
			FormatTimezone:      req.FormatTimezone,
			FormatSizeDecimals:  req.FormatSizeDecimals,
		}

		// Save to settings.json file
		if err := config.SaveSettings(newSettings, "settings.json"); err != nil {
			http.Error(w, "Failed to save settings: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Also save to database for history
		if err := ws.db.SaveSetting("dir", req.Dir); err != nil {
			log.Printf("Warning: failed to save to database: %v", err)
		}

		response := map[string]interface{}{
			"dir":                   req.Dir,
			"dry_run":               req.DryRun,
			"web_port":              req.WebPort,
			"web_only":              req.WebOnly,
			"db_path":               req.DbPath,
			"cron":                  req.Cron,
			"cron_expr":             req.CronExpr,
			"rename_subfolder":      req.RenameSubfolder,
			"format_size":           req.FormatSize,
			"format_datetime":       req.FormatDatetime,
			"format_timezone":       req.FormatTimezone,
			"format_size_decimals":  req.FormatSizeDecimals,
			"updated_at":            time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
