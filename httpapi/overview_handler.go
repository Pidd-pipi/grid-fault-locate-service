package httpapi

import (
	"net/http"
)

// getOverview GET /api/overview：配网总览（线路状态 + 故障事件 + 长时停电关注）。
func (a *App) getOverview(w http.ResponseWriter, r *http.Request) error {
	writeOK(w, a.overview.GetOverview())
	return nil
}
