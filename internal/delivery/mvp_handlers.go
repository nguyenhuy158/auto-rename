package delivery

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"strconv"
)

func (ws *WebServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	page, err := template.ParseFiles("template/settings.html")
	if err != nil {
		http.Error(w, "Template unavailable", http.StatusInternalServerError)
		return
	}
	_ = page.Execute(w, nil)
}

func (ws *WebServer) handleAPISettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Edit settings.json and restart the service", http.StatusMethodNotAllowed)
		return
	}
	settings := ws.scanner.Configuration()
	writeJSON(w, map[string]any{
		"dir":              settings.Dir,
		"quarantine_dir":   settings.QuarantineDir,
		"extensions":       settings.Extensions,
		"dry_run":          settings.DryRun,
		"web_port":         settings.WebPort,
		"web_only":         settings.WebOnly,
		"cron":             settings.Cron,
		"cron_expr":        settings.CronExpr,
		"rename_subfolder": settings.RenameSubfolder,
	})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}

func nonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

func readJSON(r *http.Request, value any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(value)
}

func requirePOST(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodPost {
		return true
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	return false
}

func pageQuery(r *http.Request, defaultSize int) (int, int) {
	page, pageSize := 1, defaultSize
	if value, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && value > 0 && value <= 1000000 {
		page = value
	}
	if value, err := strconv.Atoi(r.URL.Query().Get("pageSize")); err == nil && value > 0 {
		pageSize = min(value, 100)
	}
	return page, pageSize
}

func (ws *WebServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := ws.db.Ping(); err != nil {
		http.Error(w, "D1 unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]any{"status": "ok"})
}

func (ws *WebServer) handleDuplicatesPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	page, err := template.ParseFiles("template/duplicates.html")
	if err != nil {
		http.Error(w, "Template unavailable", http.StatusInternalServerError)
		return
	}
	if err := page.Execute(w, nil); err != nil {
		return
	}
}

func (ws *WebServer) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	page, pageSize := pageQuery(r, 20)
	search := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	runs, err := ws.db.ListRuns(page, pageSize, search, sortBy, order)
	if err != nil {
		http.Error(w, "D1 unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]any{"current": ws.scanner.Status(), "recent": nonNil(runs.Rows), "total": runs.Total, "page": page, "pageSize": pageSize})
}

func (ws *WebServer) handleDuplicates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]any{"groups": nonNil(ws.scanner.DuplicateGroups())})
}

func (ws *WebServer) handleDuplicateScan(w http.ResponseWriter, r *http.Request) {
	if !requirePOST(w, r) {
		return
	}
	id, err := ws.scanner.TriggerDuplicateScan()
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, map[string]any{"run_id": id})
}

func (ws *WebServer) handleDuplicateQuarantine(w http.ResponseWriter, r *http.Request) {
	if !requirePOST(w, r) {
		return
	}
	var input struct {
		SHA256  string   `json:"sha256"`
		Paths   []string `json:"paths"`
		All     bool     `json:"all"`
		Confirm bool     `json:"confirm"`
	}
	if err := readJSON(r, &input); err != nil || !input.Confirm {
		http.Error(w, "Explicit confirmation required", http.StatusBadRequest)
		return
	}
	id, err := ws.scanner.QuarantineDuplicates(input.SHA256, input.Paths, input.All)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, map[string]any{"run_id": id})
}

func (ws *WebServer) handleDuplicateQuarantineAll(w http.ResponseWriter, r *http.Request) {
	if !requirePOST(w, r) {
		return
	}
	var input struct {
		Confirm bool `json:"confirm"`
	}
	if err := readJSON(r, &input); err != nil || !input.Confirm {
		http.Error(w, "Explicit confirmation required", http.StatusBadRequest)
		return
	}
	id, err := ws.scanner.QuarantineAllDuplicates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, map[string]any{"run_id": id})
}

func (ws *WebServer) handleQuarantine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	page, pageSize := pageQuery(r, 20)
	actions, err := ws.db.ListQuarantines(page, pageSize, r.URL.Query().Get("search"), r.URL.Query().Get("sort"), r.URL.Query().Get("order"))
	if err != nil {
		http.Error(w, "D1 unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]any{"actions": nonNil(actions.Rows), "total": actions.Total, "page": page, "pageSize": pageSize})
}

func (ws *WebServer) handleQuarantineRestore(w http.ResponseWriter, r *http.Request) {
	if !requirePOST(w, r) {
		return
	}
	var input struct {
		ID int `json:"id"`
	}
	if err := readJSON(r, &input); err != nil || input.ID <= 0 {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}
	if err := ws.scanner.RestoreQuarantine(input.ID); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, map[string]any{"restored": true})
}

func (ws *WebServer) handleRecordRestore(w http.ResponseWriter, r *http.Request) {
	if !requirePOST(w, r) {
		return
	}
	var input struct {
		ID int `json:"id"`
	}
	if err := readJSON(r, &input); err != nil || input.ID <= 0 {
		http.Error(w, "Invalid record", http.StatusBadRequest)
		return
	}
	if err := ws.scanner.RestoreName(input.ID); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, map[string]any{"restored": true})
}
