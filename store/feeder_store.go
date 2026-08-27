package store

import (
	"sort"

	"example.com/grid-fault-locate-service/domain"
)

// CreateFeeder 新增线路。
func (s *Store) CreateFeeder(f *domain.Feeder) error {
	if f.ID == "" {
		return domain.Invalidf("feeder id is required")
	}
	return s.mutate(func() error {
		if _, ok := s.feeders[f.ID]; ok {
			return domain.Conflictf("feeder %s already exists", f.ID)
		}
		s.feeders[f.ID] = f
		return nil
	})
}

// UpdateFeeder 更新线路（整体替换）。
func (s *Store) UpdateFeeder(f *domain.Feeder) error {
	if f.ID == "" {
		return domain.Invalidf("feeder id is required")
	}
	return s.mutate(func() error {
		if _, ok := s.feeders[f.ID]; !ok {
			return domain.NotFoundf("feeder %s not found", f.ID)
		}
		s.feeders[f.ID] = f
		return nil
	})
}

// GetFeeder 查询线路。
func (s *Store) GetFeeder(id string) (*domain.Feeder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.feeders[id]
	if !ok {
		return nil, domain.NotFoundf("feeder %s not found", id)
	}
	return f, nil
}

// ListFeeders 列出全部线路；结果按 ID 升序，保证确定性。
func (s *Store) ListFeeders() []*domain.Feeder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.Feeder, 0, len(s.feeders))
	for _, f := range s.feeders {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// DeleteFeeder 删除线路（调用方需先校验无关联资源）。
func (s *Store) DeleteFeeder(id string) error {
	return s.mutate(func() error {
		if _, ok := s.feeders[id]; !ok {
			return domain.NotFoundf("feeder %s not found", id)
		}
		delete(s.feeders, id)
		return nil
	})
}
