// Package service 实现领域业务逻辑：拓扑维护、信号采集、
// 故障定位推理、隔离/复电流转、停电统计与长时停电扫描。
package service

import (
	"time"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/store"
)

// AuditService 操作审计日志服务：所有关键业务动作统一留痕。
type AuditService struct {
	store *store.Store
}

// NewAuditService 构造审计服务。
func NewAuditService(st *store.Store) *AuditService {
	return &AuditService{store: st}
}

// Record 记录一条审计日志。
// requestID 由 HTTP 中间件注入，可为空（内部任务调用）。
func (s *AuditService) Record(requestID string, action domain.AuditAction,
	entityType, entityID, operator, detail string) error {
	entry := domain.NewAuditEntry(s.store.NextID("AU"), requestID, action,
		entityType, entityID, operator, detail, time.Now())
	return s.store.AppendAudit(entry)
}

// List 列出审计日志（新→旧），limit<=0 表示不限。
func (s *AuditService) List(limit int) []*domain.AuditEntry {
	return s.store.ListAudits(limit)
}
