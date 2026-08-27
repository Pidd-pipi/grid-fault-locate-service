package httpapi

import (
	"net/http"
	"testing"
)

func TestBug005ReportSignalNotFound404(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	resp, _ := doJSON(t, h, http.MethodPost, "/api/indicators/FI-missing/signal", `{"status":"triggered"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("上报不存在的指示器应 404，实际 %d", resp.StatusCode)
	}
}

func TestBug005UpdateIndicatorNotFound404(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	resp, _ := doJSON(t, h, http.MethodPut, "/api/indicators/FI-missing", `{"name":"x","position":0.5}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("更新不存在的指示器应 404，实际 %d", resp.StatusCode)
	}
}

func TestBug005CreateIndicatorInvalid400(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	resp, _ := doJSON(t, h, http.MethodPost, "/api/indicators", `{"name":"x","sectionId":"SEC-missing"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("在不存在的区段上创建指示器应 400，实际 %d", resp.StatusCode)
	}
}
