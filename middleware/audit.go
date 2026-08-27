package middleware

import (
	"net/http"
	"strings"
	"time"

	"example.com/grid-fault-locate-service/domain"
)

// AuditService 审计写入接口（由 service.AuditService 实现）。
type AuditService interface {
	Record(requestID string, action domain.AuditAction, entityType, entityID, operator, detail string) error
}

// Audit 中间件：对 /api/* 业务请求记录 HTTP 审计（含状态码与耗时）。
// 跳过静态资源、健康检查与审计列表自身，避免噪音与循环写入。
func Audit(audit AuditService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") ||
			strings.HasPrefix(r.URL.Path, "/api/healthz") ||
			strings.HasPrefix(r.URL.Path, "/api/audit") {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		detail := r.Method + " " + r.URL.Path + " -> " + http.StatusText(rec.status)
		_ = audit.Record(GetRequestID(r.Context()), domain.AuditHTTPRequest, "http", r.URL.Path, "",
			detail+" ("+time.Since(start).Round(time.Millisecond).String()+")")
	})
}
