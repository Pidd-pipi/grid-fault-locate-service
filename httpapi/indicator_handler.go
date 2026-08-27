package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/middleware"
	"example.com/grid-fault-locate-service/service"
)

// listIndicators GET /api/indicators：指示器列表，
// 支持 ?feederId=&sectionId=&suspicious=&triggered= 过滤及 limit/offset 分页。
func (a *App) listIndicators(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	onlySuspicious, err := parseOptionalBool(q, "suspicious")
	if err != nil {
		return err
	}
	onlyTriggered, err := parseOptionalBool(q, "triggered")
	if err != nil {
		return err
	}
	limit, offset, err := parseLimitOffset(q)
	if err != nil {
		return err
	}
	page, total := paginate(a.signals.ListIndicators(q.Get("feederId"), q.Get("sectionId"), onlySuspicious, onlyTriggered), limit, offset)
	writeOKPage(w, page, total)
	return nil
}

// createIndicator POST /api/indicators：在区段上新增指示器。
func (a *App) createIndicator(w http.ResponseWriter, r *http.Request) error {
	var in service.IndicatorInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	ind, err := a.signals.CreateIndicator(in, operatorOf(r), middleware.GetRequestID(r.Context()))
	if err != nil {
		return fmt.Errorf("create indicator: %w", err)
	}
	writeCreated(w, ind)
	return nil
}

// getIndicator GET /api/indicators/{id}：指示器详情。
func (a *App) getIndicator(w http.ResponseWriter, r *http.Request) error {
	ind, err := a.signals.GetIndicator(r.PathValue("id"))
	if err != nil {
		return err
	}
	writeOK(w, ind)
	return nil
}

// updateIndicator PUT /api/indicators/{id}：更新名称/位置。
func (a *App) updateIndicator(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Name     string  `json:"name"`
		Position float64 `json:"position"`
	}
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	ind, err := a.signals.UpdateIndicator(r.PathValue("id"), in.Name, in.Position, operatorOf(r), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeOK(w, ind)
	return nil
}

// deleteIndicator DELETE /api/indicators/{id}：删除指示器。
func (a *App) deleteIndicator(w http.ResponseWriter, r *http.Request) error {
	if err := a.signals.DeleteIndicator(r.PathValue("id"), operatorOf(r), middleware.GetRequestID(r.Context())); err != nil {
		return err
	}
	writeOK(w, map[string]string{"id": r.PathValue("id"), "deleted": "true"})
	return nil
}

// signalInput 信号上报入参。
type signalInput struct {
	Status     domain.IndicatorStatus `json:"status"`
	ReportedAt *time.Time             `json:"reportedAt"`
	Source     string                 `json:"source"`
}

// reportSignal POST /api/indicators/{id}/signal：指示器信号上报（翻牌/复位）。
func (a *App) reportSignal(w http.ResponseWriter, r *http.Request) error {
	var in signalInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	var at time.Time
	if in.ReportedAt != nil {
		at = *in.ReportedAt
	}
	ind, err := a.signals.ReportSignal(r.PathValue("id"), in.Status, at, operatorOf(r), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeOK(w, ind)
	return nil
}

// flagSuspicious POST /api/indicators/{id}/suspicious：标记/清除可疑。
func (a *App) flagSuspicious(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Suspicious bool   `json:"suspicious"`
		Reason     string `json:"reason"`
	}
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	ind, err := a.signals.FlagSuspicious(r.PathValue("id"), in.Suspicious, in.Reason, operatorOf(r), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeOK(w, ind)
	return nil
}
