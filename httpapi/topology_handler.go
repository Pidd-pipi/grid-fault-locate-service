package httpapi

import (
	"net/http"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/middleware"
	"example.com/grid-fault-locate-service/service"
)

// getTopology GET /api/feeders/{id}/topology：拓扑数据（线路+开关+区段+指示器）。
func (a *App) getTopology(w http.ResponseWriter, r *http.Request) error {
	tp, err := a.topology.GetTopology(r.PathValue("id"))
	if err != nil {
		return err
	}
	writeOK(w, tp)
	return nil
}

// addSwitch POST /api/feeders/{id}/switches：新增开关节点。
func (a *App) addSwitch(w http.ResponseWriter, r *http.Request) error {
	var in service.SwitchInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	sw, err := a.topology.AddSwitch(r.PathValue("id"), in, operatorOf(r), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeCreated(w, sw)
	return nil
}

// updateSwitch PUT /api/feeders/{id}/switches/{switchId}：更新开关。
func (a *App) updateSwitch(w http.ResponseWriter, r *http.Request) error {
	var in service.SwitchInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	sw, err := a.topology.UpdateSwitch(r.PathValue("id"), r.PathValue("switchId"), in, operatorOf(r), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeOK(w, sw)
	return nil
}

// toggleSwitch POST /api/feeders/{id}/switches/{switchId}/toggle：分合闸。
func (a *App) toggleSwitch(w http.ResponseWriter, r *http.Request) error {
	sw, err := a.topology.ToggleSwitch(r.PathValue("id"), r.PathValue("switchId"), operatorOf(r), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeOK(w, sw)
	return nil
}

// removeSwitch DELETE /api/feeders/{id}/switches/{switchId}：删除开关。
func (a *App) removeSwitch(w http.ResponseWriter, r *http.Request) error {
	if err := a.topology.RemoveSwitch(r.PathValue("id"), r.PathValue("switchId"), operatorOf(r), middleware.GetRequestID(r.Context())); err != nil {
		return err
	}
	writeOK(w, map[string]string{"id": r.PathValue("switchId"), "deleted": "true"})
	return nil
}

// addSection POST /api/feeders/{id}/sections：新增区段（拓扑校验）。
func (a *App) addSection(w http.ResponseWriter, r *http.Request) error {
	var in service.SectionInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	sec, err := a.topology.AddSection(r.PathValue("id"), in, operatorOf(r), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeCreated(w, sec)
	return nil
}

// updateSection PUT /api/feeders/{id}/sections/{sectionId}：更新区段。
func (a *App) updateSection(w http.ResponseWriter, r *http.Request) error {
	var in service.SectionInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	sec, err := a.topology.UpdateSection(r.PathValue("id"), r.PathValue("sectionId"), in, operatorOf(r), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeOK(w, sec)
	return nil
}

// removeSection DELETE /api/feeders/{id}/sections/{sectionId}：删除区段（连通性校验）。
func (a *App) removeSection(w http.ResponseWriter, r *http.Request) error {
	if err := a.topology.RemoveSection(r.PathValue("id"), r.PathValue("sectionId"), operatorOf(r), middleware.GetRequestID(r.Context())); err != nil {
		return err
	}
	writeOK(w, map[string]string{"id": r.PathValue("sectionId"), "deleted": "true"})
	return nil
}

// operatorOf 从查询参数读取操作人，缺省 anonymous。
func operatorOf(r *http.Request) string {
	if op := r.URL.Query().Get("operator"); op != "" {
		return op
	}
	return "anonymous"
}
