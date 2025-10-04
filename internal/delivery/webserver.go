// Web server delivery layer for auto-rename
package delivery

import (
	"auto-rename/internal/infrastructure"
	"encoding/json"
	"html/template"
	"net/http"
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
	http.HandleFunc("/", ws.handleIndex)
	http.HandleFunc("/logs", ws.handleLogs)
	http.HandleFunc("/records", ws.handleRecord)
	http.HandleFunc("/api/records", ws.handleAPIRecords)
	http.HandleFunc("/api/stats", ws.handleAPIStats)
	return http.ListenAndServe(":"+ws.webPort, nil)
}

func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	println("handleIndex:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))
	tmpl, err := template.ParseFiles("template/index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (ws *WebServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	println("handleLogs:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))
	tmpl, err := template.ParseFiles("template/logs.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (ws *WebServer) handleRecord(w http.ResponseWriter, r *http.Request) {
	println("handleRecord:", r.URL.Path, "IP:", r.RemoteAddr, "Time:", time.Now().Format("2006-01-02 15:04:05"))
	tmpl, err := template.ParseFiles("template/records.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
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
