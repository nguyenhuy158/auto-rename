package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
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
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Auto-Rename Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #007bff; padding-bottom: 10px; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin: 30px 0; }
        .stat-card { background: #f8f9fa; padding: 20px; border-radius: 6px; text-align: center; border-left: 4px solid #007bff; }
        .stat-number { font-size: 2em; font-weight: bold; color: #007bff; }
        .stat-label { color: #666; margin-top: 5px; }
        .nav { margin: 20px 0; }
        .nav a { display: inline-block; padding: 10px 20px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; margin-right: 10px; }
        .nav a:hover { background: #0056b3; }
        .info { background: #e7f3ff; padding: 20px; border-radius: 6px; margin: 20px 0; border-left: 4px solid #007bff; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔄 Auto-Rename Dashboard</h1>
        
        <div class="info">
            <strong>Welcome!</strong> This dashboard shows the history of file rename operations performed by the auto-rename tool.
        </div>

        <div class="nav">
            <a href="/records">📋 View All Records</a>
            <a href="/api/records">📊 API - Records</a>
            <a href="/api/stats">📈 API - Statistics</a>
        </div>

        <div class="stats" id="stats">
            <div class="stat-card">
                <div class="stat-number" id="total">-</div>
                <div class="stat-label">Total Operations</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="successful">-</div>
                <div class="stat-label">Successful</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="failed">-</div>
                <div class="stat-label">Failed</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="recent">-</div>
                <div class="stat-label">Last 24h</div>
            </div>
        </div>
    </div>

    <script>
        fetch('/api/stats')
            .then(response => response.json())
            .then(data => {
                document.getElementById('total').textContent = data.total_records || 0;
                document.getElementById('successful').textContent = data.successful_renames || 0;
                document.getElementById('failed').textContent = data.failed_renames || 0;
                document.getElementById('recent').textContent = data.recent_activity || 0;
            })
            .catch(error => console.error('Error loading stats:', error));
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(tmpl))
}

func (ws *WebServer) handleRecordsPage(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	records, err := ws.db.GetAllRecords()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get records: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>File Rename Records</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { max-width: 1400px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #007bff; padding-bottom: 10px; }
        .search { margin: 20px 0; }
        .search input { padding: 10px; width: 300px; border: 1px solid #ddd; border-radius: 4px; }
        .search button { padding: 10px 20px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; margin-left: 10px; }
        .search button:hover { background: #0056b3; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f8f9fa; font-weight: bold; }
        tr:hover { background-color: #f5f5f5; }
        .success { color: #28a745; font-weight: bold; }
        .failed { color: #dc3545; font-weight: bold; }
        .nav { margin: 20px 0; }
        .nav a { display: inline-block; padding: 10px 20px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; margin-right: 10px; }
        .nav a:hover { background: #0056b3; }
        .file-name { max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .path { max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #666; font-size: 0.9em; }
        .timestamp { font-size: 0.9em; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📋 File Rename Records</h1>
        
        <div class="nav">
            <a href="/">🏠 Home</a>
            <a href="/api/records">📊 JSON Data</a>
        </div>

        <div class="search">
            <input type="text" id="searchInput" placeholder="Search by original filename...">
            <button onclick="searchRecords()">🔍 Search</button>
            <button onclick="loadAllRecords()">📋 Show All</button>
        </div>

        <table id="recordsTable">
            <thead>
                <tr>
                    <th>ID</th>
                    <th>Original Name</th>
                    <th>New Name (UUID)</th>
                    <th>File Path</th>
                    <th>Size</th>
                    <th>Status</th>
                    <th>Renamed At</th>
                    <th>Error</th>
                </tr>
            </thead>
            <tbody id="recordsBody">
            </tbody>
        </table>
    </div>

    <script>
        function loadAllRecords() {
            fetch('/api/records')
                .then(response => response.json())
                .then(data => displayRecords(data))
                .catch(error => {
                    console.error('Error loading records:', error);
                    document.getElementById('recordsBody').innerHTML = '<tr><td colspan="8">Error loading records</td></tr>';
                });
        }

        function searchRecords() {
            const query = document.getElementById('searchInput').value;
            if (!query.trim()) {
                loadAllRecords();
                return;
            }

            fetch('/api/records/search?q=' + encodeURIComponent(query))
                .then(response => response.json())
                .then(data => displayRecords(data))
                .catch(error => {
                    console.error('Error searching records:', error);
                    document.getElementById('recordsBody').innerHTML = '<tr><td colspan="8">Error searching records</td></tr>';
                });
        }

        function displayRecords(records) {
            const tbody = document.getElementById('recordsBody');
            if (!records || records.length === 0) {
                tbody.innerHTML = '<tr><td colspan="8">No records found</td></tr>';
                return;
            }

            tbody.innerHTML = records.map(record => {
                const status = record.success ? '<span class="success">✅ Success</span>' : '<span class="failed">❌ Failed</span>';
                const errorMsg = record.error_msg || '';
                const fileSize = record.file_size ? formatFileSize(record.file_size) : '-';
                const renamedAt = new Date(record.renamed_at).toLocaleString();
                
                return ` + "`" + `
                    <tr>
                        <td>${record.id}</td>
                        <td class="file-name" title="${record.original_name}">${record.original_name}</td>
                        <td class="file-name" title="${record.new_name}">${record.new_name}</td>
                        <td class="path" title="${record.file_path}">${record.file_path}</td>
                        <td>${fileSize}</td>
                        <td>${status}</td>
                        <td class="timestamp">${renamedAt}</td>
                        <td>${errorMsg}</td>
                    </tr>
                ` + "`" + `;
            }).join('');
        }

        function formatFileSize(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        // Load records on page load
        loadAllRecords();

        // Enable search on Enter key
        document.getElementById('searchInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                searchRecords();
            }
        });
    </script>
</body>
</html>`

	t, err := template.New("records").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, map[string]interface{}{
		"Records": records,
		"Page":    page,
	})
}
