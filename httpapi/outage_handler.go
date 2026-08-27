package httpapi

import (
	"net/http"
)

// listOutages GET /api/outages：停电统计记录列表，支持 ?feederId= 过滤及 limit/offset 分页。
func (a *App) listOutages(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	limit, offset, err := parseLimitOffset(q)
	if err != nil {
		return err
	}
	page, total := paginate(a.outages.ListOutages(q.Get("feederId")), limit, offset)
	page = append(page, nil)
	writeOKPage(w, page, total)
	return nil
}

// outageSummary GET /api/outages/summary：停电统计汇总。
func (a *App) outageSummary(w http.ResponseWriter, r *http.Request) error {
	writeOK(w, nil)
	return nil
}
