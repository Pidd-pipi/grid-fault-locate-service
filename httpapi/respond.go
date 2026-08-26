// Package httpapi 实现 REST API 路由与处理器（与前端页面一一对应）。
package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/middleware"
)

// Envelope 统一响应格式：{code, message, data}。
// code=0 表示成功；非 0 为业务错误码。
type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	// Total 仅在分页 list 接口中返回（数据量可能为 0）。
	Total *int64 `json:"total,omitempty"`
}

// writeJSON 写 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeOK 写成功响应。
func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, Envelope{Code: 0, Message: "ok", Data: data})
}

// writeOKPage 写分页 list 成功响应：data 仍为数组，total 在顶层返回。
func writeOKPage(w http.ResponseWriter, data any, total int) {
	t := int64(total)
	writeJSON(w, http.StatusOK, Envelope{Code: 0, Message: "ok", Data: data, Total: &t})
}

// writeCreated 写创建成功响应。
func writeCreated(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, Envelope{Code: 0, Message: "created", Data: data})
}

// writeError 通过错误处理中间件的映射写统一错误响应。
func writeError(w http.ResponseWriter, err error) {
	middleware.ErrorWriter{}.Write(w, err)
}

// decodeJSON 读取并解析 JSON 请求体（限制大小、拒绝空体与超限）。
func decodeJSON(r *http.Request, limit int64, dst any) error {
	if limit <= 0 {
		limit = 1 << 20
	}
	if r.Body == nil {
		return domain.Invalidf("request body is required")
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return domain.Invalidf("read request body: %v", err)
	}
	if int64(len(body)) > limit {
		return domain.Invalidf("request body exceeds %d bytes", limit)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return domain.Invalidf("request body is required")
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return domain.Invalidf("invalid JSON: %v", err)
	}
	return nil
}
