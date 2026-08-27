package httpapi

import (
	"net/http"
	"testing"
)

func TestBug009AuditInvalidLimitBadRequest(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	resp, _ := doJSON(t, h, http.MethodGet, "/api/audit?limit=abc", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("非法 limit 应 400，实际 %d", resp.StatusCode)
	}
}

func TestBug009HealthzOK(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	resp, _ := doJSON(t, h, http.MethodGet, "/healthz", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz 应 200，实际 %d", resp.StatusCode)
	}
}
