package service

import (
	"context"
	"log/slog"
	"time"

	"example.com/grid-fault-locate-service/config"
	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/store"
)

// LongOutageSweeper 长时停电扫描：周期扫描超阈值未复电事件，
// 标记为长时停电关注并留痕（触达 service → store → 总览）。
type LongOutageSweeper struct {
	store     *store.Store
	faults    *FaultService
	audit     *AuditService
	threshold time.Duration
	interval  time.Duration
}

// NewLongOutageSweeper 构造扫描器。
func NewLongOutageSweeper(st *store.Store, faults *FaultService, audit *AuditService, cfg config.Config) *LongOutageSweeper {
	return &LongOutageSweeper{
		store:     st,
		faults:    faults,
		audit:     audit,
		threshold: cfg.LongOutageThreshold(),
		interval:  cfg.SweepInterval,
	}
}

// RunOnce 执行一次扫描，返回本次新标记的长时停电事件 ID。
func (s *LongOutageSweeper) RunOnce(now time.Time) ([]string, error) {
	var flagged []string
	for _, event := range s.store.ListActiveFaults() {
		if !event.Status.IsActive() {
			continue
		}
		elapsed := now.Sub(event.LocatedAt)
		if elapsed > s.threshold && !event.LongOutage {
			if err := s.faults.MarkLongOutage(event.ID, now); err != nil {
				return flagged, err
			}
			flagged = append(flagged, event.ID)
			_ = s.audit.Record("", domain.AuditLongOutageScan, "fault", event.ID, "system",
				"long outage detected: located at "+event.LocatedAt.Format(time.RFC3339)+", elapsed "+
					s.elapsedStr(elapsed))
		}
	}
	return flagged, nil
}

// Start 启动周期扫描（启动即执行一次，随后按 interval 周期执行），
// 随 ctx 取消而退出。
func (s *LongOutageSweeper) Start(ctx context.Context) {
	if _, err := s.RunOnce(time.Now()); err != nil {
		slog.Error("sweeper initial scan failed", "error", err)
	}
	ticker := time.NewTicker(s.interval)
	for {
		select {
		case now := <-ticker.C:
			flagged, err := s.RunOnce(now)
			if err != nil {
				slog.Error("sweeper scan failed", "error", err)
				continue
			}
			if len(flagged) > 0 {
				slog.Info("sweeper flagged long-outage faults", "count", len(flagged), "fault_ids", flagged)
			}
		}
	}
}

func (s *LongOutageSweeper) elapsedStr(d time.Duration) string {
	return d.Round(time.Minute).String()
}
