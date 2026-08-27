package httpapi

import (
	"io/fs"
	"net/http"
	"strings"

	"example.com/grid-fault-locate-service/config"
	"example.com/grid-fault-locate-service/middleware"
	"example.com/grid-fault-locate-service/service"
	"example.com/grid-fault-locate-service/store"
)

// App 聚合全部依赖，构建路由。
type App struct {
	cfg      config.Config
	store    *store.Store
	topology *service.TopologyService
	signals  *service.SignalService
	faults   *service.FaultService
	outages  *service.OutageService
	overview *service.OverviewService
	audit    *service.AuditService
	sweeper  *service.LongOutageSweeper
	webFS    fs.FS
}

// SetSweeper 注入长时停电扫描器（由 main 构造）。
func (a *App) SetSweeper(s *service.LongOutageSweeper) {
	a.sweeper = s
}

// NewApp 构造应用。
func NewApp(cfg config.Config, st *store.Store, webFS fs.FS) *App {
	audit := service.NewAuditService(st)
	topology := service.NewTopologyService(st, audit)
	signals := service.NewSignalService(st, audit)
	locate := service.NewLocateService(st, topology, audit)
	outages := service.NewOutageService(st, audit)
	faults := service.NewFaultService(st, locate, outages, audit)
	overview := service.NewOverviewService(st, outages, faults)
	return &App{
		cfg:      cfg,
		store:    st,
		topology: topology,
		signals:  signals,
		faults:   faults,
		outages:  outages,
		overview: overview,
		audit:    audit,
		webFS:    webFS,
	}
}

// handlerFunc 处理器签名：返回 error 时统一走错误处理中间件。
type handlerFunc func(w http.ResponseWriter, r *http.Request) error

// handle 注册路由并包装错误处理。
func (a *App) handle(mux *http.ServeMux, pattern string, h handlerFunc) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			writeError(w, err)
		}
	})
}

// Handler 构建完整 HTTP 处理器（路由 + 中间件链）。
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()

	// 健康检查
	a.handle(mux, "GET /healthz", a.healthz)
	a.handle(mux, "GET /api/healthz", a.healthz)
	a.handle(mux, "GET /readyz", a.readyz)
	a.handle(mux, "GET /api/readyz", a.readyz)

	// 线路
	a.handle(mux, "GET /api/feeders", a.listFeeders)
	a.handle(mux, "POST /api/feeders", a.createFeeder)
	a.handle(mux, "GET /api/feeders/{id}", a.getFeeder)
	a.handle(mux, "PUT /api/feeders/{id}", a.updateFeeder)
	a.handle(mux, "DELETE /api/feeders/{id}", a.deleteFeeder)

	// 拓扑
	a.handle(mux, "GET /api/feeders/{id}/topology", a.getTopology)
	a.handle(mux, "POST /api/feeders/{id}/switches", a.addSwitch)
	a.handle(mux, "PUT /api/feeders/{id}/switches/{switchId}", a.updateSwitch)
	a.handle(mux, "POST /api/feeders/{id}/switches/{switchId}/toggle", a.toggleSwitch)
	a.handle(mux, "DELETE /api/feeders/{id}/switches/{switchId}", a.removeSwitch)
	a.handle(mux, "POST /api/feeders/{id}/sections", a.addSection)
	a.handle(mux, "PUT /api/feeders/{id}/sections/{sectionId}", a.updateSection)
	a.handle(mux, "DELETE /api/feeders/{id}/sections/{sectionId}", a.removeSection)

	// 指示器
	a.handle(mux, "GET /api/indicators", a.listIndicators)
	a.handle(mux, "POST /api/indicators", a.createIndicator)
	a.handle(mux, "GET /api/indicators/{id}", a.getIndicator)
	a.handle(mux, "PUT /api/indicators/{id}", a.updateIndicator)
	a.handle(mux, "DELETE /api/indicators/{id}", a.deleteIndicator)
	a.handle(mux, "POST /api/indicators/{id}/signal", a.reportSignal)
	a.handle(mux, "POST /api/indicators/{id}/suspicious", a.flagSuspicious)

	// 故障
	a.handle(mux, "GET /api/faults", a.listFaults)
	a.handle(mux, "GET /api/faults/{id}", a.getFault)
	a.handle(mux, "POST /api/faults/locate", a.locateFault)
	a.handle(mux, "POST /api/faults/{id}/repair", a.repairFault)
	a.handle(mux, "POST /api/faults/{id}/isolate", a.isolateFault)
	a.handle(mux, "POST /api/faults/{id}/restore", a.restoreFault)
	a.handle(mux, "POST /api/faults/{id}/archive", a.archiveFault)

	// 停电统计
	a.handle(mux, "GET /api/outages", a.listOutages)
	a.handle(mux, "GET /api/outages/summary", a.outageSummary)

	// 总览与审计
	a.handle(mux, "GET /api/overview", a.getOverview)
	a.handle(mux, "GET /api/audit", a.listAudit)

	// 长时停电扫描（手动触发，便于演示/验证）。
	a.handle(mux, "POST /api/admin/long-outage-scan", a.triggerScan)

	// 内嵌前端静态资源（web/）。
	sub, err := fs.Sub(a.webFS, "web")
	if err != nil {
		panic("embed web failed: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, notFoundError(r.URL.Path))
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	var handler http.Handler = mux
	handler = middleware.Audit(a.audit, handler)
	handler = middleware.RequestLog(handler)
	handler = middleware.Recover(handler)
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.RequestID(handler)
	return handler
}
