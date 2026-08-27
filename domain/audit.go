package domain

import "time"

// AuditAction 审计动作分类常量。
type AuditAction string

const (
	// AuditFeederCreate 线路创建。
	AuditFeederCreate AuditAction = "feeder.create"
	// AuditFeederUpdate 线路更新。
	AuditFeederUpdate AuditAction = "feeder.update"
	// AuditFeederDelete 线路删除。
	AuditFeederDelete AuditAction = "feeder.delete"
	// AuditSwitchAdd 开关新增。
	AuditSwitchAdd AuditAction = "switch.add"
	// AuditSwitchUpdate 开关更新（分合闸等）。
	AuditSwitchUpdate AuditAction = "switch.update"
	// AuditSwitchRemove 开关删除。
	AuditSwitchRemove AuditAction = "switch.remove"
	// AuditSectionAdd 区段新增。
	AuditSectionAdd AuditAction = "section.add"
	// AuditSectionUpdate 区段更新。
	AuditSectionUpdate AuditAction = "section.update"
	// AuditSectionRemove 区段删除。
	AuditSectionRemove AuditAction = "section.remove"
	// AuditIndicatorCreate 指示器新增。
	AuditIndicatorCreate AuditAction = "indicator.create"
	// AuditIndicatorUpdate 指示器更新。
	AuditIndicatorUpdate AuditAction = "indicator.update"
	// AuditIndicatorDelete 指示器删除。
	AuditIndicatorDelete AuditAction = "indicator.delete"
	// AuditIndicatorSignal 指示器信号上报。
	AuditIndicatorSignal AuditAction = "indicator.signal"
	// AuditIndicatorSuspicious 指示器可疑标记。
	AuditIndicatorSuspicious AuditAction = "indicator.suspicious"
	// AuditFaultLocate 故障定位。
	AuditFaultLocate AuditAction = "fault.locate"
	// AuditFaultRepair 开始抢修。
	AuditFaultRepair AuditAction = "fault.repair"
	// AuditFaultIsolate 隔离确认。
	AuditFaultIsolate AuditAction = "fault.isolate"
	// AuditFaultRestore 复电完成。
	AuditFaultRestore AuditAction = "fault.restore"
	// AuditFaultArchive 归档。
	AuditFaultArchive AuditAction = "fault.archive"
	// AuditOutageCreate 停电记录生成（复电联动生成，避免与 fault.restore 重复留痕）。
	AuditOutageCreate AuditAction = "outage.create"
	// AuditLongOutageScan 长时停电扫描。
	AuditLongOutageScan AuditAction = "sweeper.long_outage"
	// AuditHTTPRequest HTTP 请求审计（中间件写入）。
	AuditHTTPRequest AuditAction = "http.request"
)

// AuditEntry 操作审计日志条目。
type AuditEntry struct {
	ID         string      `json:"id"`
	RequestID  string      `json:"requestId"`
	Action     AuditAction `json:"action"`
	EntityType string      `json:"entityType"`
	EntityID   string      `json:"entityId"`
	Operator   string      `json:"operator"`
	Detail     string      `json:"detail"`
	CreatedAt  time.Time   `json:"createdAt"`
}

// NewAuditEntry 构造审计条目。
func NewAuditEntry(id, requestID string, action AuditAction, entityType, entityID, operator, detail string, at time.Time) *AuditEntry {
	return &AuditEntry{
		ID:         id,
		RequestID:  requestID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Operator:   operator,
		Detail:     detail,
		CreatedAt:  at,
	}
}
