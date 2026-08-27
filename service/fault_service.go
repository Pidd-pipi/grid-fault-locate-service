package service

import (
	"sort"
	"time"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/store"
)

// FaultService 故障事件生命周期：定位建单、抢修、隔离、复电、归档。
type FaultService struct {
	store  *store.Store
	locate *LocateService
	outage *OutageService
	audit  *AuditService
}

// NewFaultService 构造故障服务。
func NewFaultService(st *store.Store, locate *LocateService, outage *OutageService, audit *AuditService) *FaultService {
	return &FaultService{store: st, locate: locate, outage: outage, audit: audit}
}

// FaultFilter 故障事件列表过滤条件。
type FaultFilter struct {
	Status   domain.FaultStatus
	FeederID string
	// LongOutageOnly 仅长时停电关注事件。
	LongOutageOnly bool
}

// ListFaults 列出故障事件（按定位时间倒序）。
func (s *FaultService) ListFaults(filter FaultFilter) []*domain.FaultEvent {
	list := s.store.ListFaults(filter.Status, filter.FeederID)
	if filter.LongOutageOnly {
		out := make([]*domain.FaultEvent, 0, len(list))
		for _, f := range list {
			if f.LongOutage && f.Status.IsActive() {
				out = append(out, f)
			}
		}
		list = out
	}
	sort.Slice(list, func(i, j int) bool { return list[i].LocatedAt.After(list[j].LocatedAt) })
	return list
}

// GetFault 查询故障事件。
func (s *FaultService) GetFault(id string) (*domain.FaultEvent, error) {
	return s.store.GetFault(id)
}

// LocateAndCreateEvent 执行定位推理并创建故障事件（状态 located）。
func (s *FaultService) LocateAndCreateEvent(in LocateInput, operator, requestID string) (*domain.FaultEvent, *LocateResult, error) {
	result, err := s.locate.Locate(in)
	if err != nil {
		return nil, nil, err
	}
	feeder, err := s.store.GetFeeder(in.FeederID)
	if err != nil {
		return nil, nil, err
	}
	triggered := make([]string, 0, len(result.Triggered))
	for _, t := range result.Triggered {
		triggered = append(triggered, t.IndicatorID)
	}
	suspicious := make([]string, 0, len(result.Suspicious))
	for _, sp := range result.Suspicious {
		suspicious = append(suspicious, sp.IndicatorID)
	}
	event := domain.NewFaultEvent(s.store.NextID("FE"), feeder.ID, feeder.Name,
		result.PrimarySectionID, candidateIDs(result.Candidates), triggered, suspicious,
		result.Evidence, operator, time.Now())
	if err := s.store.CreateFault(event); err != nil {
		return nil, nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditFaultLocate, "fault", event.ID, operator,
		"locate fault on feeder "+feeder.Name+", primary section "+result.PrimarySectionID)
	return event, result, nil
}

func candidateIDs(cs []CandidateSection) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.SectionID)
	}
	return out
}

// StartRepair 开始抢修：located → repairing。
func (s *FaultService) StartRepair(id, operator, note, requestID string) (*domain.FaultEvent, error) {
	event, err := s.store.GetFault(id)
	if err != nil {
		return nil, err
	}
	if err := event.ApplyRepair(operator, note, time.Now()); err != nil {
		return nil, err
	}
	if err := s.store.UpdateFault(event); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditFaultRepair, "fault", id, operator, "start repair: "+note)
	return event, nil
}

// Isolate 隔离区段操作确认：
//   - 事件必须处于 located/repairing；
//   - 隔离区段必须属于事件线路（跨线路区段拒绝，避免误分他线开关）；
//   - 联动：区段两端开关分闸、区段标记隔离，事件进入 repairing。
func (s *FaultService) Isolate(id, operator, sectionID, note, requestID string) (*domain.FaultEvent, error) {
	event, err := s.store.GetFault(id)
	if err != nil {
		return nil, err
	}
	if sectionID == "" {
		return nil, domain.Invalidf("sectionId is required for isolation")
	}
	sec, err := s.store.GetSection(sectionID)
	if err != nil {
		return nil, err
	}
	if sec.FeederID != event.FeederID {
		return nil, domain.Conflictf("section %s does not belong to fault feeder %s", sectionID, event.FeederID)
	}

	if sec.Isolated {
		return nil, domain.Conflictf("section %s is already isolated", sectionID)
	}
	if err := event.ApplyIsolate(sectionID, operator, note, time.Now()); err != nil {
		return nil, err
	}
	// 联动：两端开关分闸 + 区段隔离标记。
	if err := s.openBoundarySwitches(sec); err != nil {
		return nil, err
	}
	sec.Isolated = true
	sec.Touch(time.Now())
	if err := s.store.UpdateSection(sec); err != nil {
		return nil, err
	}
	if err := s.store.UpdateFault(event); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditFaultIsolate, "fault", id, operator,
		"isolate section "+sectionID+" ("+note+")")
	return event, nil
}

// openBoundarySwitches 将区段两端开关分闸（隔离联动）。
func (s *FaultService) openBoundarySwitches(sec *domain.FeederSection) error {
	for _, swID := range []string{sec.UpstreamSwitchID, sec.DownstreamSwitchID} {
		sw, err := s.store.GetSwitch(swID)
		if err != nil {
			return err
		}
		if sw.Status != domain.SwitchOpen {
			sw.Status = domain.SwitchOpen
			sw.Touch(time.Now())
			if err := s.store.UpdateSwitch(sw); err != nil {
				return err
			}
		}
	}
	return nil
}

// closeBoundarySwitches 将区段两端开关合闸（复电联动）。
func (s *FaultService) closeBoundarySwitches(sec *domain.FeederSection) error {
	for _, swID := range []string{sec.UpstreamSwitchID, sec.DownstreamSwitchID} {
		sw, err := s.store.GetSwitch(swID)
		if err != nil {
			return err
		}
		if sw.Status != domain.SwitchClosed {
			sw.Status = domain.SwitchClosed
			sw.Touch(time.Now())
			if err := s.store.UpdateSwitch(sw); err != nil {
				return err
			}
		}
	}
	return nil
}

// Restore 复电完成：repairing → restored。
// 前置条件：必须已完成隔离确认（ErrNotIsolated），复电后联动合闸、
// 清除区段隔离/候选标记并生成停电统计记录。
func (s *FaultService) Restore(id, operator, note, requestID string) (*domain.FaultEvent, error) {
	event, err := s.store.GetFault(id)
	if err != nil {
		return nil, err
	}
	if err := event.ApplyRestore(operator, note, time.Now()); err != nil {
		return nil, err
	}
	// 联动：隔离区段两端合闸、清除隔离与候选标记。
	if sec, err := s.store.GetSection(event.IsolatedSectionID); err == nil {
		if err := s.closeBoundarySwitches(sec); err != nil {
			return nil, err
		}
		sec.Isolated = false
		sec.IsCandidate = false
		sec.Touch(time.Now())
		if err := s.store.UpdateSection(sec); err != nil {
			return nil, err
		}
	}
	// 清除其余候选区段的候选标记。
	for _, cid := range event.CandidateSectionIDs {
		if cid == event.IsolatedSectionID {
			continue
		}
		if sec, err := s.store.GetSection(cid); err == nil {
			sec.IsCandidate = false
			sec.Touch(time.Now())
			_ = s.store.UpdateSection(sec)
		}
	}
	if err := s.store.UpdateFault(event); err != nil {
		return nil, err
	}
	// 生成停电统计记录。
	if _, err := s.outage.CreateFromRestore(event); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditFaultRestore, "fault", id, operator, "restore power: "+note)
	return event, nil
}

// Archive 归档：restored → archived。
// 必须经过复电闭环后才能归档；未复电（located/repairing）直接归档会被状态机拒绝。
func (s *FaultService) Archive(id, operator, requestID string) (*domain.FaultEvent, error) {
	event, err := s.store.GetFault(id)
	if err != nil {
		return nil, err
	}
	if err := event.ApplyArchive(operator, time.Now()); err != nil {
		return nil, err
	}
	if err := s.store.UpdateFault(event); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditFaultArchive, "fault", id, operator, "archive fault")
	return event, nil
}

// MarkLongOutage 标记长时停电（扫描任务调用）。
func (s *FaultService) MarkLongOutage(id string, at time.Time) error {
	event, err := s.store.GetFault(id)
	if err != nil {
		return err
	}
	if !event.Status.IsActive() {
		return nil
	}
	event.MarkLongOutage(at)
	return s.store.UpdateFault(event)
}
