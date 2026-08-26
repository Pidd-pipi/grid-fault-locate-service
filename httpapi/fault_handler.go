package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/middleware"
	"example.com/grid-fault-locate-service/service"
)

// listFaults GET /api/faults：故障事件列表，支持 ?status=&feederId=&longOutage=true 及 limit/offset 分页。
func (a *App) listFaults(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	status := domain.FaultStatus(q.Get("status"))
	if status != "" && !status.Valid() {
		return domain.Invalidf("status %q is invalid", status)
	}
	longOutageOnly, err := parseOptionalBool(q, "longOutage")
	if err != nil {
		return err
	}
	limit, offset, err := parseLimitOffset(q)
	if err != nil {
		return err
	}
	filter := service.FaultFilter{
		Status:         status,
		FeederID:       q.Get("feederId"),
		LongOutageOnly: longOutageOnly,
	}
	page, total := paginate(a.faults.ListFaults(filter), limit, offset)
	writeOKPage(w, page, total)
	return nil
}

// getFault GET /api/faults/{id}：故障事件详情。
func (a *App) getFault(w http.ResponseWriter, r *http.Request) error {
	f, err := a.faults.GetFault(r.PathValue("id"))
	if err != nil {
		return fmt.Errorf("get fault: %v", err)
	}
	writeOK(w, f)
	return nil
}

// locateFault POST /api/faults/locate：故障定位推理并建单。
func (a *App) locateFault(w http.ResponseWriter, r *http.Request) error {
	var in service.LocateInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	event, result, err := a.faults.LocateAndCreateEvent(in, operatorOf(r), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeCreated(w, map[string]any{"event": event, "locate": result})
	return nil
}

// actionInput 隔离/复电/抢修/归档通用入参。
type actionInput struct {
	Operator  string `json:"operator"`
	SectionID string `json:"sectionId"`
	Note      string `json:"note"`
}

func (in *actionInput) operator() string {
	if in.Operator != "" {
		return in.Operator
	}
	return "anonymous"
}

// repairFault POST /api/faults/{id}/repair：开始抢修。
func (a *App) repairFault(w http.ResponseWriter, r *http.Request) error {
	var in actionInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	f, err := a.faults.StartRepair(r.PathValue("id"), in.operator(), in.Note, middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeOK(w, f)
	return nil
}

// isolateFault POST /api/faults/{id}/isolate：隔离区段操作确认。
func (a *App) isolateFault(w http.ResponseWriter, r *http.Request) error {
	var in actionInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	f, err := a.faults.Isolate(r.PathValue("id"), in.operator(), in.SectionID, in.Note, middleware.GetRequestID(r.Context()))
	if err != nil {
		return fmt.Errorf("isolate fault: %v", err)
	}
	writeOK(w, f)
	return nil
}

// restoreFault POST /api/faults/{id}/restore：复电完成。
func (a *App) restoreFault(w http.ResponseWriter, r *http.Request) error {
	var in actionInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	f, err := a.faults.Restore(r.PathValue("id"), in.operator(), in.Note, middleware.GetRequestID(r.Context()))
	if err != nil {
		return fmt.Errorf("restore fault: %v", err)
	}
	writeOK(w, f)
	return nil
}

// archiveFault POST /api/faults/{id}/archive：归档。
func (a *App) archiveFault(w http.ResponseWriter, r *http.Request) error {
	var in actionInput
	if err := decodeJSON(r, a.cfg.RequestBodyLimit, &in); err != nil {
		return domain.Invalidf("invalid request body: %v", err)
	}
	f, err := a.faults.Archive(r.PathValue("id"), in.operator(), middleware.GetRequestID(r.Context()))
	if err != nil {
		return err
	}
	writeOK(w, f)
	return nil
}

// triggerScan POST /api/admin/long-outage-scan：手动触发长时停电扫描。
func (a *App) triggerScan(w http.ResponseWriter, r *http.Request) error {
	if a.sweeper == nil {
		return domain.Invalidf("sweeper not configured")
	}
	flagged, err := a.sweeper.RunOnce(nowFunc())
	if err != nil {
		return err
	}
	writeOK(w, map[string]any{"flagged": flagged, "count": len(flagged)})
	return nil
}

// nowFunc 可替换的时间源（测试注入）。
var nowFunc = func() time.Time { return time.Now() }
