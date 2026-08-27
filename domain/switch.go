package domain

import "time"

// SwitchStatus 开关分合状态（前后端保持一致，前端定义于 web/enums.js）。
type SwitchStatus string

const (
	// SwitchClosed 开关合闸（合）。
	SwitchClosed SwitchStatus = "closed"
	// SwitchOpen 开关分闸（分）。
	SwitchOpen SwitchStatus = "open"
)

// Valid 校验开关状态枚举值。
func (s SwitchStatus) Valid() bool {
	switch s {
	case SwitchClosed, SwitchOpen:
		return true
	}
	return false
}

// SwitchType 开关类型（前后端保持一致，前端定义于 web/enums.js）。
type SwitchType string

const (
	// SwitchTypeSectionalizer 分段开关。
	SwitchTypeSectionalizer SwitchType = "sectionalizer"
	// SwitchTypeTie 联络开关。
	SwitchTypeTie SwitchType = "tie"
	// SwitchTypeFeederOutlet 出线开关（每条线路有且只有一个，位于首端）。
	SwitchTypeFeederOutlet SwitchType = "feeder_outlet"
)

// Valid 校验开关类型枚举值。
func (t SwitchType) Valid() bool {
	switch t {
	case SwitchTypeSectionalizer, SwitchTypeTie, SwitchTypeFeederOutlet:
		return true
	}
	return false
}

// SwitchNode 开关节点实体，是区段的端点。
type SwitchNode struct {
	ID         string       `json:"id"`
	FeederID   string       `json:"feederId"`
	Name       string       `json:"name"`
	SwitchType SwitchType   `json:"switchType"`
	Status     SwitchStatus `json:"status"`
	// Order 线路内展示顺序（1 为首端，出线开关必须为 1）。
	Order     int       `json:"order"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Validate 校验开关字段合法性。
func (s *SwitchNode) Validate() error {
	if s.Name == "" {
		return Invalidf("switch name is required")
	}
	if !s.SwitchType.Valid() {
		return Invalidf("switch type %q is invalid", s.SwitchType)
	}
	if !s.Status.Valid() {
		return Invalidf("switch status %q is invalid", s.Status)
	}
	return nil
}

// Toggle 切换开关分合状态。
func (s *SwitchNode) Toggle() {
	if s.Status == SwitchClosed {
		s.Status = SwitchOpen
	} else {
		s.Status = SwitchClosed
	}
}

// Touch 更新开关 UpdatedAt 时间戳。
func (s *SwitchNode) Touch(now time.Time) {
	s.UpdatedAt = now
}

// IsFeederOutlet 判断是否为出线开关。
func (s *SwitchNode) IsFeederOutlet() bool {
	return s.SwitchType == SwitchTypeFeederOutlet
}

// NewSwitchNode 构造开关节点实体。
func NewSwitchNode(id, feederID, name string, swType SwitchType, status SwitchStatus, order int, now time.Time) *SwitchNode {
	if status == "" {
		status = SwitchClosed
	}
	return &SwitchNode{
		ID:         id,
		FeederID:   feederID,
		Name:       name,
		SwitchType: swType,
		Status:     status,
		Order:      order,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}
