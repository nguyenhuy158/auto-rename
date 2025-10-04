package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
)

type WebServer struct {
	db   *Database
	port string
}

func NewWebServer(db *Database, port string) *WebServer {
	return &WebServer{
		db:   db,
		port: port,
	}
}

func (ws *WebServer) Start() error {
	r := mux.NewRouter()

	// API endpoints
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/records", ws.handleGetRecords).Methods("GET")
	api.HandleFunc("/records/search", ws.handleSearchRecords).Methods("GET")
	api.HandleFunc("/stats", ws.handleGetStats).Methods("GET")
	api.HandleFunc("/cron/status", ws.handleGetCronStatus).Methods("GET")

	// Web interface
	r.HandleFunc("/", ws.handleHome).Methods("GET")
	r.HandleFunc("/records", ws.handleRecordsPage).Methods("GET")

	// Static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Printf("Starting web server on port %s", ws.port)
	log.Printf("Access the web interface at: http://localhost:%s", ws.port)
	log.Printf("API endpoints available at: http://localhost:%s/api/", ws.port)

	return http.ListenAndServe(":"+ws.port, r)
}

func (ws *WebServer) handleGetRecords(w http.ResponseWriter, r *http.Request) {
	records, err := ws.db.GetAllRecords()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get records: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func (ws *WebServer) handleSearchRecords(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	records, err := ws.db.GetRecordsByOriginalName(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search records: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func (ws *WebServer) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := ws.db.GetStats()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (ws *WebServer) handleHome(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("template/index.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load template: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (ws *WebServer) handleRecordsPage(w http.ResponseWriter, r *http.Request) {
	// Page parameter retained for potential future pagination (currently unused by static template)
	if p := r.URL.Query().Get("page"); p != "" {
		// Intentionally ignored for now
		_, _ = strconv.Atoi(p) // best-effort parse to avoid unused import warning
	}

	data, err := os.ReadFile("template/records.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load template: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (ws *WebServer) handleGetCronStatus(w http.ResponseWriter, r *http.Request) {
	status := GetCronStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
