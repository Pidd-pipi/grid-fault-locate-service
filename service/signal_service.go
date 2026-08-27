package service

import (
	"fmt"
	"time"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/store"
)

// SignalService 故障指示器信号采集与可疑标记。
type SignalService struct {
	store *store.Store
	audit *AuditService
}

// NewSignalService 构造信号服务。
func NewSignalService(st *store.Store, audit *AuditService) *SignalService {
	return &SignalService{store: st, audit: audit}
}

// IndicatorInput 指示器创建入参。
type IndicatorInput struct {
	Name      string  `json:"name"`
	SectionID string  `json:"sectionId"`
	Position  float64 `json:"position"`
}

// CreateIndicator 在指定区段上新增指示器。
func (s *SignalService) CreateIndicator(in IndicatorInput, operator, requestID string) (*domain.FaultIndicator, error) {
	if in.Name == "" {
		return nil, domain.Invalidf("indicator name is required")
	}
	sec, err := s.store.GetSection(in.SectionID)
	if err != nil {
		return nil, domain.Invalidf("section: %v", err)
	}
	if in.Position < 0 {
		return nil, domain.Invalidf("indicator position must not be negative")
	}
	now := time.Now()
	ind := domain.NewFaultIndicator(s.store.NextID("FI"), sec.FeederID, sec.ID, in.Name, in.Position, now)
	if err := ind.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.CreateIndicator(ind); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditIndicatorCreate, "indicator", ind.ID, operator,
		"create indicator "+ind.Name+" on section "+sec.ID)
	return ind, nil
}

// UpdateIndicator 更新指示器名称/位置。
func (s *SignalService) UpdateIndicator(id string, name string, position float64, operator, requestID string) (*domain.FaultIndicator, error) {
	ind, err := s.store.GetIndicator(id)
	if err != nil {
		return nil, fmt.Errorf("update indicator get: %w", err)
	}
	if name != "" {
		ind.Name = name
	}
	if position >= 0 {
		ind.Position = position
	}
	ind.UpdatedAt = time.Now()
	if err := ind.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.UpdateIndicator(ind); err != nil {
		return nil, fmt.Errorf("update indicator save: %w", err)
	}
	_ = s.audit.Record(requestID, domain.AuditIndicatorUpdate, "indicator", ind.ID, operator, "update indicator "+ind.Name)
	return ind, nil
}

// DeleteIndicator 删除指示器。
func (s *SignalService) DeleteIndicator(id, operator, requestID string) error {
	ind, err := s.store.GetIndicator(id)
	if err != nil {
		return fmt.Errorf("delete indicator get: %w", err)
	}
	if err := s.store.DeleteIndicator(id); err != nil {
		return fmt.Errorf("delete indicator: %w", err)
	}
	_ = s.audit.Record(requestID, domain.AuditIndicatorDelete, "indicator", id, operator, "delete indicator "+ind.Name)
	return nil
}

// ReportSignal 指示器信号上报（翻牌 triggered / 复位 reset）。
// 复位信号自动清除可疑标记；同一区段冲突信号在定位推理时处理。
func (s *SignalService) ReportSignal(indicatorID string, status domain.IndicatorStatus, at time.Time, operator, requestID string) (*domain.FaultIndicator, error) {
	if !status.Valid() {
		return nil, domain.Invalidf("indicator status %q is invalid", status)
	}
	ind, err := s.store.GetIndicator(indicatorID)
	if err != nil {
		return nil, fmt.Errorf("report signal: %w", err)
	}
	if at.IsZero() {
		at = time.Now()
	}
	ind.ApplySignal(status, at)
	if err := s.store.UpdateIndicator(ind); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditIndicatorSignal, "indicator", ind.ID, operator,
		"indicator "+ind.Name+" reports "+string(status))
	return ind, nil
}

// FlagSuspicious 标记/清除指示器可疑状态（人工核验）。
func (s *SignalService) FlagSuspicious(indicatorID string, suspicious bool, reason string, operator, requestID string) (*domain.FaultIndicator, error) {
	ind, err := s.store.GetIndicator(indicatorID)
	if err != nil {
		return nil, err
	}
	if suspicious && reason == "" {
		reason = "manual verification required"
	}
	ind.MarkSuspicious(suspicious, reason, time.Now())
	if err := s.store.UpdateIndicator(ind); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditIndicatorSuspicious, "indicator", ind.ID, operator,
		"indicator "+ind.Name+" suspicious="+boolStr(suspicious)+" ("+reason+")")
	return ind, nil
}

// ListIndicators 列出指示器。
func (s *SignalService) ListIndicators(feederID, sectionID string, onlySuspicious bool, onlyTriggered bool) []*domain.FaultIndicator {
	list := s.store.ListIndicators(feederID, sectionID)
	out := make([]*domain.FaultIndicator, 0, len(list))
	for _, ind := range list {
		if onlySuspicious && !ind.Suspicious {
			continue
		}
		if onlyTriggered && !ind.IsTriggered() {
			continue
		}
		out = append(out, ind)
	}
	return out
}

// GetIndicator 查询指示器。
func (s *SignalService) GetIndicator(id string) (*domain.FaultIndicator, error) {
	return s.store.GetIndicator(id)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
