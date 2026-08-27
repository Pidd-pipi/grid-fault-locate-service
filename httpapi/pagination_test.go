package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListPaginationReturnsTotalAndPage(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	for i := 0; i < 3; i++ {
		resp, _ := doJSON(t, h, http.MethodPost, "/api/feeders", `{"name":"线路`+string(rune('A'+i))+`","substation":"站","voltageLevel":"10kV"}`)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create feeder failed: %d", resp.StatusCode)
		}
	}

	resp, payload := doJSON(t, h, http.MethodGet, "/api/feeders?limit=2&offset=1", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list feeders should 200, got %d", resp.StatusCode)
	}
	total, _ := payload["total"].(float64)
	if total != 3 {
		t.Fatalf("total should be 3, got %v", total)
	}
	data, ok := payload["data"].([]any)
	if !ok || len(data) != 2 {
		t.Fatalf("page should have 2 items, got %v", payload["data"])
	}
}

func TestListPaginationRejectsInvalidParams(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	for _, path := range []string{
		"/api/feeders?limit=-1",
		"/api/feeders?offset=abc",
		"/api/faults?longOutage=yes",
	} {
		resp, _ := doJSON(t, h, http.MethodGet, path, "")
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s should return 400 for invalid params, got %d", path, resp.StatusCode)
		}
	}
}

func TestSecurityHeadersAndReadyz(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	for _, path := range []string{"/api/healthz", "/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s should be 200, got %d", path, rec.Code)
		}
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" ||
			rec.Header().Get("X-Frame-Options") != "DENY" ||
			rec.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Fatalf("security headers missing for %s: %v", path, rec.Header())
		}
		if path == "/api/healthz" && rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("API should set Cache-Control no-store, got %q", rec.Header().Get("Cache-Control"))
		}
		if rec.Header().Get("X-Request-Id") == "" {
			t.Fatalf("X-Request-Id missing for %s", path)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz should 200, got %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["data"].(map[string]any)["status"] != "ready" {
		t.Fatalf("readyz data mismatch: %v", payload)
	}
}
