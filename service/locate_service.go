package service

import (
	"fmt"
	"sort"
	"time"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/store"
)

// LocateService 故障定位推理：综合指示器信号 + 拓扑关系定位候选故障区段。
type LocateService struct {
	store    *store.Store
	topology *TopologyService
	audit    *AuditService
}

// NewLocateService 构造定位服务。
func NewLocateService(st *store.Store, topology *TopologyService, audit *AuditService) *LocateService {
	return &LocateService{store: st, topology: topology, audit: audit}
}

// LocateInput 定位请求入参。
type LocateInput struct {
	FeederID string `json:"feederId"`
}

// TriggeredSignal 参与定位的翻牌信号。
type TriggeredSignal struct {
	IndicatorID   string    `json:"indicatorId"`
	IndicatorName string    `json:"indicatorName"`
	SectionID     string    `json:"sectionId"`
	ReportedAt    time.Time `json:"reportedAt"`
}

// SuspiciousSignal 可疑（孤立翻牌）信号，已排除出定位依据。
type SuspiciousSignal struct {
	IndicatorID   string `json:"indicatorId"`
	IndicatorName string `json:"indicatorName"`
	SectionID     string `json:"sectionId"`
	Reason        string `json:"reason"`
}

// CandidateSection 候选故障区段。
type CandidateSection struct {
	SectionID          string    `json:"sectionId"`
	SectionName        string    `json:"sectionName"`
	UpstreamSwitchID   string    `json:"upstreamSwitchId"`
	DownstreamSwitchID string    `json:"downstreamSwitchId"`
	TriggeredAt        time.Time `json:"triggeredAt"`
	Order              int       `json:"order"`
}

// LocateResult 定位推理结果。
type LocateResult struct {
	FeederID         string             `json:"feederId"`
	PrimarySectionID string             `json:"primarySectionId"`
	Candidates       []CandidateSection `json:"candidates"`
	Triggered        []TriggeredSignal  `json:"triggered"`
	Suspicious       []SuspiciousSignal `json:"suspicious"`
	Evidence         string             `json:"evidence"`
}

// Locate 执行故障定位推理。
//
// 核心规则（领域规则 2/3/6）：
//  1. 排除可疑指示器，仅以非可疑信号参与推理；
//  2. 从出线开关向下游遍历区段，记录最后一个翻牌指示器；
//  3. 找到第一个未翻牌（复位）指示器：若与翻牌同区段 → 候选即该区段
//     （翻牌与未翻牌之间）；否则候选为「最后一个翻牌区段 .. 首个复位区段-1」；
//  4. 遍历结束仍无复位信号 → 候选为最后一个翻牌区段（故障在线路末端）；
//  5. 多候选按信号时间排序输出，主候选为翻牌信号最新者；
//  6. 孤立翻牌（所在区段与相邻区段均无其他翻牌且不在候选区）标记可疑，不参与定位。
func (s *LocateService) Locate(in LocateInput) (*LocateResult, error) {
	if in.FeederID == "" {
		return nil, domain.Invalidf("feederId is required")
	}
	if _, err := s.store.GetFeeder(in.FeederID); err != nil {
		return nil, err
	}
	tp, err := s.topology.GetTopology(in.FeederID)
	if err != nil {
		return nil, err
	}
	// 区段按下游顺序排列。
	sectionByID := make(map[string]*domain.FeederSection, len(tp.Sections))
	for _, sec := range tp.Sections {
		sectionByID[sec.ID] = sec
	}
	ordered := make([]*domain.FeederSection, 0, len(tp.SectionOrder))
	for _, id := range tp.SectionOrder {
		if sec, ok := sectionByID[id]; ok {
			ordered = append(ordered, sec)
		}
	}

	indicators := s.store.ListIndicators(in.FeederID, "")
	bySection := make(map[string][]*domain.FaultIndicator)
	for _, ind := range indicators {
		bySection[ind.SectionID] = append(bySection[ind.SectionID], ind)
	}

	result := &LocateResult{FeederID: in.FeederID}
	var allTriggered []TriggeredSignal

	// 第一遍：统计参与推理的信号，同时收集已标记可疑的翻牌信号。
	for _, ind := range indicators {
		if !ind.IsTriggered() {
			continue
		}
		ts := TriggeredSignal{IndicatorID: ind.ID, IndicatorName: ind.Name, SectionID: ind.SectionID, ReportedAt: ind.ReportedAt}
		if ind.Suspicious {
			result.Suspicious = append(result.Suspicious, SuspiciousSignal{
				IndicatorID: ind.ID, IndicatorName: ind.Name, SectionID: ind.SectionID, Reason: ind.SuspiciousReason,
			})
			continue
		}
		allTriggered = append(allTriggered, ts)
	}
	if len(allTriggered) == 0 {
		if len(result.Suspicious) > 0 {
			return nil, domain.ErrSuspiciousOnly
		}
		return nil, domain.ErrNoFaultSignal
	}
	result.Triggered = allTriggered

	// 第二遍：先定位「最后一个翻牌区段」，再找其后「第一个未翻牌区段」。
	lastTriggeredIdx := -1
	for i, sec := range ordered {
		if latest := latestTriggeredOn(bySection[sec.ID]); !latest.IsZero() {
			lastTriggeredIdx = i
		}
	}
	if lastTriggeredIdx < 0 || lastTriggeredIdx >= len(ordered) {
		return nil, domain.ErrNoFaultSignal
	}

	firstResetIdx := -1
	sameSectionConflict := false
	for i := lastTriggeredIdx; i < len(ordered); i++ {
		if sectionHasReset(bySection[ordered[i].ID]) {
			firstResetIdx = i
			sameSectionConflict = i == lastTriggeredIdx
			break
		}
	}

	candidateRangeStart := lastTriggeredIdx
	candidateRangeEnd := lastTriggeredIdx // inclusive
	switch {
	case sameSectionConflict:
		// 翻牌与未翻牌同区段：候选即该区段。
		candidateRangeStart, candidateRangeEnd = lastTriggeredIdx, lastTriggeredIdx
	case firstResetIdx > lastTriggeredIdx:
		// 最后一个翻牌区段 与 第一个未翻牌区段 之间的区段。
		candidateRangeEnd = firstResetIdx - 1
	default:
		// 无下游未翻牌信号：故障在线路末端，候选为最后一个翻牌区段。
		candidateRangeStart, candidateRangeEnd = lastTriggeredIdx, lastTriggeredIdx
	}

	// 组装候选区段。
	for i := candidateRangeStart; i <= candidateRangeEnd && i < len(ordered); i++ {
		sec := ordered[i]
		result.Candidates = append(result.Candidates, CandidateSection{
			SectionID:          sec.ID,
			SectionName:        sec.Name,
			UpstreamSwitchID:   sec.UpstreamSwitchID,
			DownstreamSwitchID: sec.DownstreamSwitchID,
			TriggeredAt:        latestTriggeredOn(bySection[sec.ID]),
			Order:              i,
		})
	}
	// 主候选：最后一个翻牌指示器所在区段。
	result.PrimarySectionID = ordered[candidateRangeStart].ID
	// 多候选按信号时间升序排序输出（无翻牌信号的候选排后）。
	sort.Slice(result.Candidates, func(i, j int) bool {
		li, lj := result.Candidates[i].TriggeredAt.IsZero(), result.Candidates[j].TriggeredAt.IsZero()
		if li != lj {
			return !li
		}
		if !result.Candidates[i].TriggeredAt.Equal(result.Candidates[j].TriggeredAt) {
			return result.Candidates[i].TriggeredAt.Before(result.Candidates[j].TriggeredAt)
		}
		return result.Candidates[i].Order < result.Candidates[j].Order
	})

	// 第三遍：孤立翻牌识别（不在候选区、且相邻区段均无翻牌）。
	candidateSet := make(map[string]bool, len(result.Candidates))
	for _, c := range result.Candidates {
		candidateSet[c.SectionID] = true
	}
	sectionIDs := make([]string, 0, len(ordered))
	for _, sec := range ordered {
		sectionIDs = append(sectionIDs, sec.ID)
	}
	sectionHasTriggered := make(map[string]bool)
	for _, t := range allTriggered {
		sectionHasTriggered[t.SectionID] = true
	}
	for _, t := range allTriggered {
		if candidateSet[t.SectionID] {
			continue
		}
		isolated := true
		for _, nb := range domain.NeighborSectionIDs(tp.Sections, t.SectionID) {
			if sectionHasTriggered[nb] {
				isolated = false
				break
			}
		}
		if isolated {
			reason := "isolated triggered signal, no triggered signal on adjacent sections"
			result.Suspicious = append(result.Suspicious, SuspiciousSignal{
				IndicatorID: t.IndicatorID, IndicatorName: t.IndicatorName, SectionID: t.SectionID, Reason: reason,
			})
			// 持久化可疑标记并留痕。
			if ind, err := s.store.GetIndicator(t.IndicatorID); err == nil {
				ind.MarkSuspicious(true, reason, time.Now())
				_ = s.store.UpdateIndicator(ind)
				_ = s.audit.Record("", domain.AuditIndicatorSuspicious, "indicator", ind.ID, "system",
					"auto-flag suspicious: "+reason)
			}
		}
	}
	// 去重可疑列表（按 IndicatorID）。
	result.Suspicious = dedupSuspicious(result.Suspicious)

	result.Evidence = s.buildEvidence(in.FeederID, tp, ordered, result)
	return result, nil
}

// sectionHasReset 判断区段上是否存在非可疑的复位信号。
func sectionHasReset(inds []*domain.FaultIndicator) bool {
	for _, ind := range inds {
		if ind.Suspicious || ind.IsTriggered() {
			continue
		}
		return true
	}
	return false
}

// latestTriggeredOn 返回区段上非可疑翻牌信号的最新上报时间。
func latestTriggeredOn(inds []*domain.FaultIndicator) time.Time {
	var latest time.Time
	for _, ind := range inds {
		if ind.Suspicious || !ind.IsTriggered() {
			continue
		}
		if ind.ReportedAt.After(latest) {
			latest = ind.ReportedAt
		}
	}
	return latest
}

// dedupSuspicious 按 IndicatorID 去重可疑信号。
func dedupSuspicious(list []SuspiciousSignal) []SuspiciousSignal {
	seen := make(map[string]bool, len(list))
	out := make([]SuspiciousSignal, 0, len(list))
	for _, s := range list {
		if seen[s.IndicatorID] {
			continue
		}
		seen[s.IndicatorID] = true
		out = append(out, s)
	}
	return out
}

// buildEvidence 生成定位依据的可读摘要。
func (s *LocateService) buildEvidence(feederID string, tp *Topology, ordered []*domain.FeederSection, result *LocateResult) string {
	outlet := "出线开关"
	for _, sw := range tp.Switches {
		if sw.IsFeederOutlet() {
			outlet = sw.Name
			break
		}
	}
	sigParts := make([]string, 0, len(result.Triggered))
	for _, t := range result.Triggered {
		sigParts = append(sigParts, fmt.Sprintf("%s@%s 翻牌(%s)", t.IndicatorName, t.SectionID, t.ReportedAt.Format("15:04:05")))
	}
	candParts := make([]string, 0, len(result.Candidates))
	for _, c := range result.Candidates {
		candParts = append(candParts, c.SectionName)
	}
	evidence := fmt.Sprintf("从%s向下游遍历，翻牌信号: %s → 候选故障区段: %s",
		outlet, joinStr(sigParts, "、"), joinStr(candParts, "、"))
	if len(result.Suspicious) > 0 {
		evidence += fmt.Sprintf("；可疑信号%d个已排除，需人工核验", len(result.Suspicious))
	}
	return evidence
}

func joinStr(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	if out == "" {
		return "-"
	}
	return out
}
