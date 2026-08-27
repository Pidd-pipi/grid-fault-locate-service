// Package middleware 提供 HTTP 横切关注点：
// trace id 注入、操作审计、统一错误/panic 处理。
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type ctxKey int

const requestIDKey ctxKey = 0

// RequestIDHeader 响应头中 trace id 的字段名。
const RequestIDHeader = "X-Request-Id"

// RequestID 中间件：为每个请求注入 trace id 并透传到下游上下文与响应头。
// 优先复用上游网关传入的 X-Request-Id（保持流水号一致），缺失时再生成。
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get(RequestIDHeader)
		if rid == "" {
			rid = newRequestID()
		}
		w.Header().Set(RequestIDHeader, rid)
		ctx := context.WithValue(r.Context(), requestIDKey, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID 从上下文读取 trace id；未注入时返回空串。
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// newRequestID 生成请求级 trace id：8 字节随机数 hex 编码（16 字符），
// 足够唯一且无时钟依赖，便于与网关流水号串联。
func newRequestID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "req-" + hex.EncodeToString(b[:])
}
