package store

import (
	"example.com/grid-fault-locate-service/domain"
)

// CreateFault 新增故障事件。
func (s *Store) CreateFault(f *domain.FaultEvent) error {
	if f.ID == "" {
		return domain.Invalidf("fault id is required")
	}
	return s.mutate(func() error {
		if _, ok := s.faults[f.ID]; ok {
			return domain.Conflictf("fault %s already exists", f.ID)
		}
		s.faults[f.ID] = f
		return nil
	})
}

// UpdateFault 更新故障事件（整体替换）。
func (s *Store) UpdateFault(f *domain.FaultEvent) error {
	if f.ID == "" {
		return domain.Invalidf("fault id is required")
	}
	return s.mutate(func() error {
		if _, ok := s.faults[f.ID]; !ok {
			return domain.NotFoundf("fault %s not found", f.ID)
		}
		s.faults[f.ID] = f
		return nil
	})
}

// GetFault 查询故障事件。
func (s *Store) GetFault(id string) (*domain.FaultEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.faults[id]
	if !ok {
		return nil, domain.NotFoundf("fault %s not found", id)
	}
	return f, nil
}

// ListFaults 列出故障事件，可选按状态/线路过滤。
func (s *Store) ListFaults(status domain.FaultStatus, feederID string) []*domain.FaultEvent {
	out := make([]*domain.FaultEvent, 0, len(s.faults))
	for _, f := range s.faults {
		if status != "" && f.Status != status {
			continue
		}
		if feederID != "" && f.FeederID != feederID {
			continue
		}
		out = append(out, f)
	}
	return out
}

// DeleteFault 删除故障事件（仅供清理场景）。
func (s *Store) DeleteFault(id string) error {
	return s.mutate(func() error {
		if _, ok := s.faults[id]; !ok {
			return domain.NotFoundf("fault %s not found", id)
		}
		delete(s.faults, id)
		return nil
	})
}

// ListActiveFaults 列出仍在处置中（located/repairing）的故障事件。
func (s *Store) ListActiveFaults() []*domain.FaultEvent {
	out := make([]*domain.FaultEvent, 0, len(s.faults))
	for _, f := range s.faults {
		if f.Status.IsActive() {
			out = append(out, f)
		}
	}
	return out
}
