package domain

import "time"

// IndicatorStatus 故障指示器状态（前后端保持一致，前端定义于 web/enums.js）。
type IndicatorStatus string

const (
	// IndicatorTriggered 指示器翻牌：检测到故障电流。
	IndicatorTriggered IndicatorStatus = "triggered"
	// IndicatorReset 指示器复位：未检测到故障电流。
	IndicatorReset IndicatorStatus = "reset"
)

// Valid 校验指示器状态枚举值。
func (s IndicatorStatus) Valid() bool {
	switch s {
	case IndicatorTriggered, IndicatorReset:
		return true
	}
	return false
}

// FaultIndicator 故障指示器实体：挂接在区段上，翻牌/复位上报故障电流信号。
type FaultIndicator struct {
	ID        string          `json:"id"`
	FeederID  string          `json:"feederId"`
	SectionID string          `json:"sectionId"`
	Name      string          `json:"name"`
	Status    IndicatorStatus `json:"status"`
	// Position 指示器在区段内的安装位置（km，距区段上游端）。
	Position         float64   `json:"position"`
	Suspicious       bool      `json:"suspicious"`
	SuspiciousReason string    `json:"suspiciousReason"`
	ReportedAt       time.Time `json:"reportedAt"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// Validate 校验指示器字段合法性。
func (i *FaultIndicator) Validate() error {
	if i.Name == "" {
		return Invalidf("indicator name is required")
	}
	if i.SectionID == "" {
		return Invalidf("indicator sectionId is required")
	}
	if i.Status != "" && !i.Status.Valid() {
		return Invalidf("indicator status %q is invalid", i.Status)
	}
	if i.Position < 0 {
		return Invalidf("indicator position must not be negative")
	}
	return nil
}

// IsTriggered 判断指示器是否处于翻牌状态。
func (i *FaultIndicator) IsTriggered() bool {
	return i.Status == IndicatorTriggered
}

// ApplySignal 应用一次信号上报：更新状态、上报时间与可疑标记。
// 复位信号自动清除可疑标记。
func (i *FaultIndicator) ApplySignal(status IndicatorStatus, at time.Time) {
	i.Status = status
	i.ReportedAt = at
	i.UpdatedAt = at
	if status == IndicatorReset {
		i.Suspicious = false
		i.SuspiciousReason = ""
	}
}

// MarkSuspicious 设置/清除可疑标记并记录原因。
func (i *FaultIndicator) MarkSuspicious(suspicious bool, reason string, at time.Time) {
	i.Suspicious = suspicious
	i.SuspiciousReason = reason
	if !suspicious {
		i.SuspiciousReason = ""
	}
	i.UpdatedAt = at
}

// NewFaultIndicator 构造故障指示器实体。
func NewFaultIndicator(id, feederID, sectionID, name string, position float64, now time.Time) *FaultIndicator {
	return &FaultIndicator{
		ID:        id,
		FeederID:  feederID,
		SectionID: sectionID,
		Name:      name,
		Status:    IndicatorReset,
		Position:  position,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
