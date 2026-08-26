package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"example.com/grid-fault-locate-service/domain"
)

func TestRequestID_InjectsAndPropagates(t *testing.T) {
	var got string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetRequestID(r.Context())
		if w.Header().Get(RequestIDHeader) == "" {
			t.Error("response header missing")
		}
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	RequestID(inner).ServeHTTP(rec, req)
	if got == "" {
		t.Fatal("request id should be generated")
	}
	if len(got) < 8 {
		t.Fatalf("request id too short: %s", got)
	}
}

func TestRequestID_RespectsIncoming(t *testing.T) {
	var got string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetRequestID(r.Context())
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(RequestIDHeader, "req-incoming")
	RequestID(inner).ServeHTTP(httptest.NewRecorder(), req)
	if got != "req-incoming" {
		t.Fatalf("incoming request id should be preserved, got %s", got)
	}
}

type fakeAudit struct {
	records []string
}

func (f *fakeAudit) Record(_ string, action domain.AuditAction, _, _, _, detail string) error {
	f.records = append(f.records, string(action)+":"+detail)
	return nil
}

func TestAudit_SkipsStaticAndAuditsAPI(t *testing.T) {
	f := &fakeAudit{}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond)
		w.WriteHeader(http.StatusCreated)
	})
	handler := Audit(f, inner)

	// 静态路径不记录
	req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if len(f.records) != 0 {
		t.Fatalf("static path should not be audited, got %v", f.records)
	}
	// API 路径记录
	req = httptest.NewRequest(http.MethodPost, "/api/faults/locate", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if len(f.records) != 1 {
		t.Fatalf("api path should be audited, got %v", f.records)
	}
	if f.records[0] != "http.request:POST /api/faults/locate -> Created (1ms)" && !contains(f.records[0], "Created") {
		t.Fatalf("unexpected audit entry: %s", f.records[0])
	}
}

func TestRecover_PanicTo500JSON(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	Recover(inner).ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic should map to 500, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatal("panic response body should not be empty")
	}
}

func TestMapError(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{domain.ErrNotFound, http.StatusNotFound},
		{domain.ErrConflict, http.StatusConflict},
		{domain.ErrStateTransition, http.StatusConflict},
		{domain.ErrNotIsolated, http.StatusConflict},
		{domain.ErrTopologyInvalid, http.StatusBadRequest},
		{domain.ErrInvalid, http.StatusBadRequest},
		{domain.ErrNoFaultSignal, http.StatusBadRequest},
		{errors.New("unknown"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		status, _, _ := MapError(c.err)
		if status != c.status {
			t.Fatalf("MapError(%v) = %d, want %d", c.err, status, c.status)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
