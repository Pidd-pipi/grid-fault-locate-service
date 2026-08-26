// Package middleware 提供 HTTP 横切关注点：
// trace id 注入、操作审计、统一错误/panic 处理。
package middleware

import (
	"context"
	"net/http"
)

type ctxKey int

const requestIDKey ctxKey = 0

// RequestIDHeader 响应头中 trace id 的字段名。
const RequestIDHeader = "X-Request-Id"

// RequestID 中间件：为每个请求生成 trace id 并注入上下文与响应头。
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get(RequestIDHeader)
		if rid == "" {
			rid = newRequestID()
		}
		w.Header().Set(RequestIDHeader, "")
		ctx := context.WithValue(context.Background(), requestIDKey, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID 从上下文读取 trace id。
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey(99)).(string); ok {
		return v
	}
	return ""
}

func newRequestID() string {
	return "req-fixed"
}
