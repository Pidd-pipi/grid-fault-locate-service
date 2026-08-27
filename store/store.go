// Package store 实现内存仓储 + JSON 文件持久化。
// 所有仓储操作集中在 Store 上（按实体拆分到独立文件），
// 每次变更后原子写盘，可重复构建、无外部服务依赖。
package store

import (
	"fmt"
	"sync"

	"example.com/grid-fault-locate-service/domain"
)

// Store 是全部实体的内存仓储根，持有互斥锁保护并发访问。
type Store struct {
	mu sync.RWMutex

	feeders    map[string]*domain.Feeder
	switches   map[string]*domain.SwitchNode
	sections   map[string]*domain.FeederSection
	indicators map[string]*domain.FaultIndicator
	faults     map[string]*domain.FaultEvent
	outages    map[string]*domain.OutageRecord
	audits     []*domain.AuditEntry

	// seq 各实体前缀的递增序列号。
	seq map[string]int64

	// persister 非 nil 时启用 JSON 文件持久化。
	persister *Persister
}

// New 构造仓储。persistPath 为空时仅使用内存（测试用），
// 否则启动时从文件加载、每次变更后原子写盘。
func New(persistPath string) (*Store, error) {
	s := &Store{
		feeders:    make(map[string]*domain.Feeder),
		switches:   make(map[string]*domain.SwitchNode),
		sections:   make(map[string]*domain.FeederSection),
		indicators: make(map[string]*domain.FaultIndicator),
		faults:     make(map[string]*domain.FaultEvent),
		outages:    make(map[string]*domain.OutageRecord),
		audits:     make([]*domain.AuditEntry, 0, 64),
		seq:        make(map[string]int64),
	}
	if persistPath != "" {
		p, err := NewPersister(persistPath)
		if err != nil {
			return nil, err
		}
		s.persister = p
		if err := s.load(); err != nil {
			return nil, fmt.Errorf("load store from %s: %w", persistPath, err)
		}
	}
	return s, nil
}

// NextID 生成带前缀的递增 ID，如 F-001、SW-003。
func (s *Store) NextID(prefix string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq[prefix]++
	return fmt.Sprintf("%s-%03d", prefix, s.seq[prefix])
}

// PersistPath 返回当前持久化文件路径（未启用时为空）。
func (s *Store) PersistPath() string {
	if s.persister == nil {
		return ""
	}
	return s.persister.path
}

// Len 返回各实体数量（调试/健康检查用）。
func (s *Store) Len() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]int{
		"feeders":    len(s.feeders),
		"switches":   len(s.switches),
		"sections":   len(s.sections),
		"indicators": len(s.indicators),
		"faults":     len(s.faults),
		"outages":    len(s.outages),
		"audits":     len(s.audits),
	}
}

// mutate 在写锁内执行变更并持久化。
func (s *Store) mutate(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(); err != nil {
		return err
	}
	return s.persistLocked()
}

// persistLocked 调用方必须已持有写锁。
func (s *Store) persistLocked() error {
	if s.persister == nil {
		return nil
	}
	return s.persister.Save(s.snapshotLocked())
}
