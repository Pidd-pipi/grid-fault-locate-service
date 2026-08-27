package service

import (
	"sort"
	"strconv"
	"time"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/store"
)

// OutageService 停电统计：复电时生成记录并汇总。
type OutageService struct {
	store *store.Store
	audit *AuditService
}

// NewOutageService 构造停电统计服务。
func NewOutageService(st *store.Store, audit *AuditService) *OutageService {
	return &OutageService{store: st, audit: audit}
}

// CreateFromRestore 由已复电故障事件生成停电记录。
// 停电开始=定位时间，结束=复电时间，超 120 分钟标记长时停电。
func (s *OutageService) CreateFromRestore(event *domain.FaultEvent) (*domain.OutageRecord, error) {
	if event.Status != domain.FaultRestored {
		return nil, domain.Statef("outage record requires restored fault, got %q", event.Status)
	}
	rec := domain.NewOutageRecord(s.store.NextID("OR"), event, time.Now())
	if err := s.store.CreateOutage(rec); err != nil {
		return nil, err
	}
	// 复电联动生成停电记录单独留痕；不复用 fault.restore 动作，避免与
	// FaultService.Restore 的审计条目重复（同一请求刷两遍）。
	_ = s.audit.Record("", domain.AuditOutageCreate, "outage", rec.ID, event.RestoredBy,
		"create outage record, duration "+strconv.Itoa(rec.DurationMinutes)+" minutes")
	return rec, nil
}

// ListOutages 列出停电记录（按停电开始时间倒序）。
func (s *OutageService) ListOutages(feederID string) []*domain.OutageRecord {
	list := s.store.ListOutages(feederID)
	sort.Slice(list, func(i, j int) bool { return list[i].OutageStart.After(list[j].OutageStart) })
	return list
}

// Summary 停电统计汇总。
func (s *OutageService) Summary() domain.OutageSummary {
	records := s.store.ListOutages("")
	byFeeder := make(map[string]*domain.FeederOutageStat)
	totalMinutes := 0
	maxMinutes := 0
	longCount := 0
	for _, r := range records {
		totalMinutes += r.DurationMinutes
		if r.DurationMinutes > maxMinutes {
			maxMinutes = r.DurationMinutes
		}
		if r.LongOutage {
			longCount++
		}
		stat, ok := byFeeder[r.FeederID]
		if !ok {
			stat = &domain.FeederOutageStat{FeederID: r.FeederID, FeederName: r.FeederName}
			byFeeder[r.FeederID] = stat
		}
		stat.RecordCount++
		stat.TotalMinutes += r.DurationMinutes
		if r.LongOutage {
			stat.LongOutages++
		}
	}
	summary := domain.OutageSummary{
		TotalRecords:    len(records),
		TotalMinutes:    totalMinutes,
		MaxMinutes:      maxMinutes,
		LongOutageCount: longCount,
	}
	if len(records) > 0 {
		summary.AvgMinutes = float64(totalMinutes) / float64(len(records))
	}
	for _, stat := range byFeeder {
		summary.ByFeeder = append(summary.ByFeeder, *stat)
	}
	sort.Slice(summary.ByFeeder, func(i, j int) bool {
		return summary.ByFeeder[i].TotalMinutes > summary.ByFeeder[j].TotalMinutes
	})
	return summary
}
