package store

import (
	"sort"

	"example.com/grid-fault-locate-service/domain"
)

// CreateOutage 新增停电统计记录。
func (s *Store) CreateOutage(o *domain.OutageRecord) error {
	if o.ID == "" {
		return domain.Invalidf("outage id is required")
	}
	return s.mutate(func() error {
		if _, ok := s.outages[o.ID]; ok {
			return domain.Conflictf("outage %s already exists", o.ID)
		}
		s.outages[o.ID] = o
		return nil
	})
}

// GetOutage 查询停电记录。
func (s *Store) GetOutage(id string) (*domain.OutageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.outages[id]
	if !ok {
		return nil, domain.NotFoundf("outage %s not found", id)
	}
	return o, nil
}

// ListOutages 列出停电记录，可选按线路过滤；结果按 ID 升序，保证确定性。
func (s *Store) ListOutages(feederID string) []*domain.OutageRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.OutageRecord, 0, len(s.outages))
	for _, o := range s.outages {
		if feederID != "" && o.FeederID != feederID {
			continue
		}
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// DeleteOutage 删除停电记录（仅供清理场景）。
func (s *Store) DeleteOutage(id string) error {
	return s.mutate(func() error {
		if _, ok := s.outages[id]; !ok {
			return domain.NotFoundf("outage %s not found", id)
		}
		delete(s.outages, id)
		return nil
	})
}
