// Web server delivery layer for auto-rename
package delivery

import (
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
	tmpl, err := template.ParseFiles("template/index.html")
	if err != nil {
		http.Error(w, "Template index error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (ws *WebServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	println("handleLogs:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))
	tmpl, err := template.ParseFiles("template/logs.html")
	if err != nil {
		http.Error(w, "Template logs error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (ws *WebServer) handleRecord(w http.ResponseWriter, r *http.Request) {
	println("handleRecord:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))
	tmpl, err := template.ParseFiles("template/records.html")
	if err != nil {
		http.Error(w, "Template records error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (ws *WebServer) handleAPIRecords(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	page := 1
	pageSize := 20
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
	records, err := ws.db.GetFileRecordsPage(page, pageSize)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	total, err := ws.db.CountFileRecords()
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
	stats := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(stats)
}
