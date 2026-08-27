package service

import (
	"sort"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/store"
)

// OverviewService 配网总览聚合：线路状态、故障事件、长时停电关注。
type OverviewService struct {
	store  *store.Store
	outage *OutageService
	faults *FaultService
}

// NewOverviewService 构造总览服务。
func NewOverviewService(st *store.Store, outage *OutageService, faults *FaultService) *OverviewService {
	return &OverviewService{store: st, outage: outage, faults: faults}
}

// FeederSummary 单条线路在总览中的摘要。
type FeederSummary struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Substation     string              `json:"substation"`
	VoltageLevel   string              `json:"voltageLevel"`
	Status         domain.FeederStatus `json:"status"`
	SwitchCount    int                 `json:"switchCount"`
	SectionCount   int                 `json:"sectionCount"`
	IndicatorCount int                 `json:"indicatorCount"`
	ActiveFaults   int                 `json:"activeFaults"`
}

// Overview 配网总览数据。
type Overview struct {
	Feeders          []FeederSummary      `json:"feeders"`
	FeederCount      int                  `json:"feederCount"`
	SwitchCount      int                  `json:"switchCount"`
	SectionCount     int                  `json:"sectionCount"`
	IndicatorCount   int                  `json:"indicatorCount"`
	TriggeredCount   int                  `json:"triggeredCount"`
	SuspiciousCount  int                  `json:"suspiciousCount"`
	ActiveFaults     []*domain.FaultEvent `json:"activeFaults"`
	LongOutageFaults []*domain.FaultEvent `json:"longOutageFaults"`
	RecentFaults     []*domain.FaultEvent `json:"recentFaults"`
	OutageSummary    domain.OutageSummary `json:"outageSummary"`
}

// GetOverview 聚合总览数据。
func (s *OverviewService) GetOverview() Overview {
	feeders := s.store.ListFeeders()
	switches := s.store.ListSwitches("")
	sections := s.store.ListSections("")
	indicators := s.store.ListIndicators("", "")

	summary := s.outage.Summary()
	activeFaults := s.store.ListActiveFaults()
	sort.Slice(activeFaults, func(i, j int) bool { return activeFaults[i].LocatedAt.After(activeFaults[j].LocatedAt) })

	var longOutage []*domain.FaultEvent
	for _, f := range activeFaults {
		if f.LongOutage {
			longOutage = append(longOutage, f)
		}
	}

	// 近期故障：按定位时间倒序取前 5 条（含处于处置中的活跃事件）。
	recent := s.faults.ListFaults(FaultFilter{})
	if len(recent) > 5 {
		recent = recent[:5]
	}

	feederSummaries := make([]FeederSummary, 0, len(feeders))
	for _, f := range feeders {
		fs := FeederSummary{
			ID:             f.ID,
			Name:           f.Name,
			Substation:     f.Substation,
			VoltageLevel:   f.VoltageLevel,
			Status:         f.Status,
			SwitchCount:    s.store.CountSwitchesOfFeeder(f.ID),
			SectionCount:   s.store.CountSectionsOfFeeder(f.ID),
			IndicatorCount: countIndicatorsOf(indicators, f.ID),
		}
		for _, af := range activeFaults {
			if af.FeederID == f.ID {
				fs.ActiveFaults++
			}
		}
		feederSummaries = append(feederSummaries, fs)
	}
	sort.Slice(feederSummaries, func(i, j int) bool { return feederSummaries[i].ID < feederSummaries[j].ID })

	triggered := 0
	suspicious := 0
	for _, ind := range indicators {
		if ind.IsTriggered() {
			triggered++
		}
		if ind.Suspicious {
			suspicious++
		}
	}

	return Overview{
		Feeders:          feederSummaries,
		FeederCount:      len(feeders),
		SwitchCount:      len(switches),
		SectionCount:     len(sections),
		IndicatorCount:   len(indicators),
		TriggeredCount:   triggered,
		SuspiciousCount:  suspicious,
		ActiveFaults:     activeFaults,
		LongOutageFaults: longOutage,
		RecentFaults:     recent,
		OutageSummary:    summary,
	}
}

func countIndicatorsOf(indicators []*domain.FaultIndicator, feederID string) int {
	n := 0
	for _, ind := range indicators {
		if ind.FeederID == feederID {
			n++
		}
	}
	return n
}
