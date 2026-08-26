package domain

import "time"

// FaultStatus 故障事件状态机状态（前后端保持一致，前端定义于 web/enums.js）。
type FaultStatus string

const (
	// FaultLocated 已定位：定位推理完成，等待处置。
	FaultLocated FaultStatus = "located"
	// FaultRepairing 抢修中：隔离确认或开始抢修后进入。
	FaultRepairing FaultStatus = "repairing"
	// FaultRestored 已复电：隔离完成后复电成功。
	FaultRestored FaultStatus = "restored"
	// FaultArchived 已归档：复电后关闭归档。
	FaultArchived FaultStatus = "archived"
)

// Valid 校验故障状态枚举值。
func (s FaultStatus) Valid() bool {
	switch s {
	case FaultLocated, FaultRepairing, FaultRestored, FaultArchived:
		return true
	}
	return false
}

// IsActive 判断事件是否仍处于处置中（未复电/未归档）。
func (s FaultStatus) IsActive() bool {
	return s == FaultLocated || s == FaultRepairing
}

// FaultAction 故障处置动作名称，状态机迁移表以动作为索引。
type FaultAction string

const (
	// ActionLocate 定位。
	ActionLocate FaultAction = "locate"
	// ActionRepair 开始抢修。
	ActionRepair FaultAction = "repair"
	// ActionIsolate 隔离区段确认。
	ActionIsolate FaultAction = "isolate"
	// ActionRestore 复电完成。
	ActionRestore FaultAction = "restore"
	// ActionArchive 归档。
	ActionArchive FaultAction = "archive"
)

// FaultTransition 描述状态机的一条合法迁移边。
type FaultTransition struct {
	Action FaultAction
	From   FaultStatus
	To     FaultStatus
	// RequiresIsolation 迁移是否要求已完成隔离确认。
	RequiresIsolation bool
}

// FaultTransitions 是故障事件完整状态机迁移表。
// located --repair--> repairing
// located --isolate--> repairing
// repairing --isolate--> repairing（重复隔离）
// repairing --restore--> restored（前置：必须已隔离）
// restored --archive--> archived
var FaultTransitions = []FaultTransition{
	{Action: ActionRepair, From: FaultLocated, To: FaultRepairing},
	{Action: ActionIsolate, From: FaultLocated, To: FaultRepairing},
	{Action: ActionIsolate, From: FaultRepairing, To: FaultRepairing},
	{Action: ActionRestore, From: FaultRepairing, To: FaultRestored, RequiresIsolation: false},
	{Action: ActionArchive, From: FaultRestored, To: FaultArchived},
	{Action: ActionArchive, From: FaultLocated, To: FaultArchived},
	{Action: ActionArchive, From: FaultRepairing, To: FaultArchived},
}

// CanTransition 判断从当前状态执行某动作是否合法。
func (s FaultStatus) CanTransition(action FaultAction) bool {
	for _, t := range FaultTransitions {
		if t.Action == action && t.From == s {
			return true
		}
	}
	return false
}

// NextStatus 返回当前状态执行某动作后的目标状态。
func (s FaultStatus) NextStatus(action FaultAction) (FaultStatus, bool) {
	for _, t := range FaultTransitions {
		if t.Action == action && t.From == s {
			return t.To, true
		}
	}
	return "", false
}

// Transition 返回当前状态到动作对应的迁移定义。
func (s FaultStatus) Transition(action FaultAction) (FaultTransition, bool) {
	for _, t := range FaultTransitions {
		if t.Action == action && t.From == s {
			return t, true
		}
	}
	return FaultTransition{}, false
}

// FaultEvent 故障事件实体：承载定位、隔离、抢修、复电、归档全生命周期。
type FaultEvent struct {
	ID                     string      `json:"id"`
	FeederID               string      `json:"feederId"`
	FeederName             string      `json:"feederName"`
	Status                 FaultStatus `json:"status"`
	CandidateSectionIDs    []string    `json:"candidateSectionIds"`
	PrimarySectionID       string      `json:"primarySectionId"`
	TriggeredIndicatorIDs  []string    `json:"triggeredIndicatorIds"`
	SuspiciousIndicatorIDs []string    `json:"suspiciousIndicatorIds"`
	Evidence               string      `json:"evidence"`
	Operator               string      `json:"operator"`
	Note                   string      `json:"note"`
	LocatedAt              time.Time   `json:"locatedAt"`
	RepairStartedAt        time.Time   `json:"repairStartedAt"`
	IsolatedSectionID      string      `json:"isolatedSectionId"`
	IsolatedAt             time.Time   `json:"isolatedAt"`
	IsolatedBy             string      `json:"isolatedBy"`
	IsolationNote          string      `json:"isolationNote"`
	RestoredAt             time.Time   `json:"restoredAt"`
	RestoredBy             string      `json:"restoredBy"`
	RestoreNote            string      `json:"restoreNote"`
	ArchivedAt             time.Time   `json:"archivedAt"`
	ArchivedBy             string      `json:"archivedBy"`
	LongOutage             bool        `json:"longOutage"`
	LongOutageSince        time.Time   `json:"longOutageSince"`
	CreatedAt              time.Time   `json:"createdAt"`
	UpdatedAt              time.Time   `json:"updatedAt"`
}

// Isolated 判断事件是否已完成隔离确认。
func (e *FaultEvent) Isolated() bool {
	return e.IsolatedSectionID != "" && !e.IsolatedAt.IsZero()
}

// OutageDuration 返回从定位到复电的停电时长；未复电则返回 0。
func (e *FaultEvent) OutageDuration() time.Duration {
	if e.RestoredAt.IsZero() || e.LocatedAt.IsZero() {
		return 0
	}
	if e.RestoredAt.Before(e.LocatedAt) {
		return 0
	}
	return e.RestoredAt.Sub(e.LocatedAt)
}

// OutageDurationMinutes 返回停电时长分钟数。
func (e *FaultEvent) OutageDurationMinutes() int {
	return int(e.OutageDuration().Minutes())
}

// ValidateAction 校验对当前状态执行某动作是否合法，返回目标状态。
func (e *FaultEvent) ValidateAction(action FaultAction) (FaultStatus, error) {
	t, ok := e.Status.Transition(action)
	if !ok {
		return "", Statef("fault %s cannot apply action %q from status %q", e.ID, action, e.Status)
	}
	if t.RequiresIsolation && !e.Isolated() {
		return "", Statef("fault %s requires isolation confirmation before %q", e.ID, action)
	}
	return t.To, nil
}

// ApplyRepair 开始抢修：located → repairing。
func (e *FaultEvent) ApplyRepair(operator, note string, at time.Time) error {
	to, err := e.ValidateAction(ActionRepair)
	if err != nil {
		return err
	}
	e.Status = to
	e.Operator = operator
	e.Note = note
	e.RepairStartedAt = at
	e.UpdatedAt = at
	return nil
}

// ApplyIsolate 隔离确认：located/repairing → repairing，记录隔离信息。
func (e *FaultEvent) ApplyIsolate(sectionID, operator, note string, at time.Time) error {
	to, err := e.ValidateAction(ActionIsolate)
	if err != nil {
		return err
	}
	e.Status = to
	e.IsolatedSectionID = sectionID
	e.IsolatedAt = at
	e.IsolatedBy = operator
	e.IsolationNote = note
	e.Operator = operator
	e.UpdatedAt = at
	return nil
}

// ApplyRestore 复电完成：repairing → restored（必须已隔离）。
func (e *FaultEvent) ApplyRestore(operator, note string, at time.Time) error {
	to, err := e.ValidateAction(ActionRestore)
	if err != nil {
		return err
	}
	e.Status = to
	e.RestoredAt = at
	e.RestoredBy = operator
	e.RestoreNote = note
	e.Operator = operator
	e.UpdatedAt = at
	return nil
}

// ApplyArchive 归档：restored → archived。
func (e *FaultEvent) ApplyArchive(operator string, at time.Time) error {
	to, err := e.ValidateAction(ActionArchive)
	if err != nil {
		return err
	}
	e.Status = to
	e.ArchivedAt = at
	e.ArchivedBy = operator
	e.Operator = operator
	e.UpdatedAt = at
	return nil
}

// MarkLongOutage 标记为长时停电关注（由扫描任务调用，不改变状态机）。
func (e *FaultEvent) MarkLongOutage(at time.Time) {
	if !e.LongOutage {
		e.LongOutage = true
		e.LongOutageSince = at
		e.UpdatedAt = at
	}
}

// NewFaultEvent 构造已定位状态的故障事件。
func NewFaultEvent(id, feederID, feederName string, primarySectionID string,
	candidates, triggered, suspicious []string, evidence, operator string, at time.Time) *FaultEvent {
	return &FaultEvent{
		ID:                     id,
		FeederID:               feederID,
		FeederName:             feederName,
		Status:                 FaultLocated,
		PrimarySectionID:       primarySectionID,
		CandidateSectionIDs:    candidates,
		TriggeredIndicatorIDs:  triggered,
		SuspiciousIndicatorIDs: suspicious,
		Evidence:               evidence,
		Operator:               operator,
		LocatedAt:              at,
		CreatedAt:              at,
		UpdatedAt:              at,
	}
}
