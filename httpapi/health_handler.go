package httpapi

import (
	"net/http"
	"time"
)

// healthz 健康检查：GET /healthz 与 GET /api/healthz。
func (a *App) healthz(w http.ResponseWriter, r *http.Request) error {
	writeOK(w, map[string]any{
		"status":  "ok",
		"service": "grid-fault-locate-service",
		"time":    time.Now().Format(time.RFC3339),
		"counts":  a.store.Len(),
	})
	return nil
}

// readyz 就绪检查：GET /readyz 与 GET /api/readyz。
func (a *App) readyz(w http.ResponseWriter, r *http.Request) error {
	writeOK(w, map[string]any{
		"status":  "ready",
		"service": "grid-fault-locate-service",
		"time":    time.Now().Format(time.RFC3339),
	})
	return nil
}
