// grid-fault-locate-service：配电网故障定位与复电服务。
// 基于 Go 实现的全栈 Web 应用（后端服务 + go:embed 内嵌前端页面），
// 完成配网拓扑管理、故障指示器信号采集、故障区段定位、复电流程跟踪与停电统计。
package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"example.com/grid-fault-locate-service/config"
	"example.com/grid-fault-locate-service/httpapi"
	"example.com/grid-fault-locate-service/service"
	"example.com/grid-fault-locate-service/store"
)

//go:embed all:web
var webFS embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}
	initLogging(cfg.LogLevel)

	st, err := store.New(cfg.DataFile)
	if err != nil {
		slog.Error("init store failed", "error", err)
		os.Exit(1)
	}

	// 组装服务依赖。
	audit := service.NewAuditService(st)
	topology := service.NewTopologyService(st, audit)
	signals := service.NewSignalService(st, audit)
	locate := service.NewLocateService(st, topology, audit)
	outages := service.NewOutageService(st, audit)
	faults := service.NewFaultService(st, locate, outages, audit)
	bootstrap := service.NewBootstrap(st, topology, signals)
	if err := bootstrap.SeedIfEmpty(); err != nil {
		slog.Error("seed demo data failed", "error", err)
		os.Exit(1)
	}

	app := httpapi.NewApp(cfg, st, webFS)
	sweeper := service.NewLongOutageSweeper(st, faults, audit, cfg)
	app.SetSweeper(sweeper)

	// 启动长时停电扫描（周期由 SWEEP_INTERVAL 控制，默认 10 分钟）。
	sweeperCtx, stopSweeper := context.WithCancel(context.Background())
	go sweeper.Start(sweeperCtx)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 优雅退出：SIGINT/SIGTERM。
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		slog.Info("shutdown signal received")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stopSweeper()
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("server shutdown failed", "error", err)
		}
	}()

	slog.Info("grid-fault-locate-service listening",
		"addr", srv.Addr,
		"data_file", cfg.DataFile,
		"persist", cfg.Persist,
		"log_level", cfg.LogLevel)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped, port released", "port", cfg.Port)
}

func initLogging(level string) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(h))
}
