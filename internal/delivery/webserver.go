// Web server delivery layer for auto-rename
package delivery

import (
	"fmt"
	"net/http"

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
	return http.ListenAndServe(":"+ws.webPort, nil)
}

func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Auto-Rename Web Server running on port %s", ws.webPort)
}
