package middleware

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"example.com/grid-fault-locate-service/domain"
)

// errorEnvelope 统一错误响应结构。
type errorEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// MapError 将领域错误映射为 (HTTP 状态码, 业务错误码, 用户可读消息)。
func MapError(err error) (int, int, string) {
	switch {
	case err == nil:
		return http.StatusOK, 0, "ok"
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, 40401, err.Error()
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, 40901, err.Error()
	case errors.Is(err, domain.ErrStateTransition):
		return http.StatusConflict, 40902, err.Error()
	case errors.Is(err, domain.ErrNotIsolated):
		return http.StatusConflict, 40903, err.Error()
	case errors.Is(err, domain.ErrTopologyInvalid):
		return http.StatusBadRequest, 40001, err.Error()
	case errors.Is(err, domain.ErrInvalid):
		return http.StatusBadRequest, 40002, err.Error()
	case errors.Is(err, domain.ErrNoFaultSignal):
		return http.StatusBadRequest, 40003, err.Error()
	case errors.Is(err, domain.ErrSuspiciousOnly):
		return http.StatusBadRequest, 40004, err.Error()
	default:
		return http.StatusInternalServerError, 50000, err.Error()
	}
}

// ErrorWriter 将业务错误写成统一 JSON 响应。
type ErrorWriter struct{}

// Write 将错误写入响应（含状态码、业务码、消息）。
func (ErrorWriter) Write(w http.ResponseWriter, err error) {
	status, code, msg := MapError(err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Code: code, Message: msg, Data: nil})
}

// Recover 中间件：捕获 handler panic，返回统一 500 JSON 响应。
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "method", r.Method, "path", r.URL.Path, "panic", rec)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(errorEnvelope{Code: 50000, Message: "internal server error", Data: nil})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
