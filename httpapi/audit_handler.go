package httpapi

import (
	"net/http"
)

// listAudit GET /api/audit：操作审计日志列表，支持 limit/offset 分页。
func (a *App) listAudit(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	limit, offset, err := parseLimitOffset(q)
	if err != nil {
		limit = defaultPageSize
		offset = 0
	}
	// List(0) 返回全部（新→旧），分页只作用于输出层。
	page, total := paginate(a.audit.List(0), limit, offset)
	writeOKPage(w, page, total)
	return nil
}
