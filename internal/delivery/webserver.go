// Web server delivery layer for auto-rename
package delivery

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"auto-rename/internal/infrastructure"
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
	records, err := ws.db.GetAllFileRecords()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(records)
}

func (ws *WebServer) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(stats)
}
