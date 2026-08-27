package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/grid-fault-locate-service/domain"
)

type bug006Audit struct {
	requestIDs []string
}

func (f *bug006Audit) Record(requestID string, action domain.AuditAction, entityType, entityID, operator, detail string) error {
	f.requestIDs = append(f.requestIDs, requestID)
	return nil
}

func TestBug006HeaderCarriesRequestID(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	RequestID(inner).ServeHTTP(rec, req)
	if rec.Header().Get(RequestIDHeader) == "" {
		t.Fatal("响应头应带非空 X-Request-Id")
	}
}

func TestBug006GetRequestIDReturnsRid(t *testing.T) {
	var got string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = GetRequestID(r.Context()) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	RequestID(inner).ServeHTTP(httptest.NewRecorder(), req)
	if got == "" {
		t.Fatal("GetRequestID 应返回注入的 request id")
	}
}

func TestBug006ParentContextPreserved(t *testing.T) {
	type parentKey string
	const k parentKey = "parent"
	ctx := context.WithValue(context.Background(), k, "secret")
	var got any
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = r.Context().Value(k) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	RequestID(inner).ServeHTTP(httptest.NewRecorder(), req)
	if got != "secret" {
		t.Fatal("RequestID 应保留父上下文的取消/值语义")
	}
}

func TestBug006RequestIDUnique(t *testing.T) {
	a := newRequestID()
	b := newRequestID()
	if a == "" || a == b {
		t.Fatal("newRequestID 应生成唯一的非空 id")
	}
}

func TestBug006AuditCarriesRequestID(t *testing.T) {
	f := &bug006Audit{}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	req := httptest.NewRequest(http.MethodPost, "/api/feeders", nil)
	rec := httptest.NewRecorder()
	h := Audit(f, inner)
	RequestID(h).ServeHTTP(rec, req)
	if len(f.requestIDs) != 1 || f.requestIDs[0] == "" {
		t.Fatalf("审计记录应带 request_id，实际 %v", f.requestIDs)
	}
}

func TestBug006RequestLogCarriesRequestID(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	old := slog.Default()
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(old)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	RequestID(RequestLog(inner)).ServeHTTP(httptest.NewRecorder(), req)
	out := buf.String()
	if !strings.Contains(out, `"request_id":"req-`) {
		t.Fatalf("访问日志应带非空 request_id，实际 %s", out)
	}
}
