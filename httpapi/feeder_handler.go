package httpapi

import (
	"net/http"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/middleware"
	"example.com/grid-fault-locate-service/service"
)

// listFeeders GET /api/feeders：线路列表，支持 ?status= 过滤及 limit/offset 分页。
func (a *App) listFeeders(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	status := domain.FeederStatus(q.Get("status"))
	if status != "" && !status.Valid() {
		return domain.Invalidf("status %q is invalid", status)
	}
	limit, offset, err := parseLimitOffset(q)
	if err != nil {
		return err
	}
	page, total := paginate(a.topology.ListFeeders(status), limit, offset)
	writeOKPage(w, page, total)
	return nil
}

// createFeeder POST /api/feeders：新增线路。
func (a *App) createFeeder(w http.ResponseWriter, r *http.Request) error {
	var in service.FeederInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	f, err := a.topology.CreateFeeder(in, r.FormValue("operator"), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeCreated(w, f)
	return nil
}

// getFeeder GET /api/feeders/{id}：线路详情。
func (a *App) getFeeder(w http.ResponseWriter, r *http.Request) error {
	f, err := a.topology.GetFeeder(r.PathValue("id"))
	if err != nil {
		return err
	}
	writeOK(w, f)
	return nil
}

// updateFeeder PUT /api/feeders/{id}：更新线路。
func (a *App) updateFeeder(w http.ResponseWriter, r *http.Request) error {
	var in service.FeederInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	f, err := a.topology.UpdateFeeder(r.PathValue("id"), in, r.FormValue("operator"), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeOK(w, f)
	return nil
}

// deleteFeeder DELETE /api/feeders/{id}：删除线路。
func (a *App) deleteFeeder(w http.ResponseWriter, r *http.Request) error {
	if err := a.topology.DeleteFeeder(r.PathValue("id"), r.FormValue("operator"), middleware.GetRequestID(r.Context())); err != nil {
		return err
	}
	writeOK(w, map[string]string{"id": r.PathValue("id"), "deleted": "true"})
	return nil
}
