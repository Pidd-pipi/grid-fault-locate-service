package domain

import "time"

// FeederStatus 线路运行状态（前后端保持一致，前端定义于 web/enums.js）。
type FeederStatus string

const (
	// FeederActive 线路在运。
	FeederActive FeederStatus = "active"
	// FeederInactive 线路停运/退出运行。
	FeederInactive FeederStatus = "inactive"
)

// Valid 校验线路状态枚举值。
func (s FeederStatus) Valid() bool {
	switch s {
	case FeederActive, FeederInactive:
		return true
	}
	return false
}

// Feeder 配网线路实体：变电站出线到末端的一条配电馈线。
type Feeder struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Substation   string       `json:"substation"`
	VoltageLevel string       `json:"voltageLevel"`
	Status       FeederStatus `json:"status"`
	Description  string       `json:"description"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

// Validate 校验线路字段合法性。
func (f *Feeder) Validate() error {
	if f.Name == "" {
		return Invalidf("feeder name is required")
	}
	if f.Substation == "" {
		return Invalidf("feeder substation is required")
	}
	if f.VoltageLevel == "" {
		return Invalidf("feeder voltageLevel is required")
	}
	if !f.Status.Valid() {
		return Invalidf("feeder status %q is invalid", f.Status)
	}
	return nil
}

// Touch 更新线路的 UpdatedAt 时间戳。
func (f *Feeder) Touch(now time.Time) {
	f.UpdatedAt = now
}

// NewFeeder 构造线路实体并初始化时间戳与默认状态。
func NewFeeder(id, name, substation, voltageLevel string, status FeederStatus, now time.Time) *Feeder {
	if status == "" {
		status = FeederActive
	}
	return &Feeder{
		ID:           id,
		Name:         name,
		Substation:   substation,
		VoltageLevel: voltageLevel,
		Status:       status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
