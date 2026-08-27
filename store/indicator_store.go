package store

import (
	"sort"

	"example.com/grid-fault-locate-service/domain"
)

// CreateIndicator 新增故障指示器。
func (s *Store) CreateIndicator(ind *domain.FaultIndicator) error {
	if ind.ID == "" {
		return domain.Invalidf("indicator id is required")
	}
	return s.mutate(func() error {
		if _, ok := s.indicators[ind.ID]; ok {
			return domain.Conflictf("indicator %s already exists", ind.ID)
		}
		s.indicators[ind.ID] = ind
		return nil
	})
}

// UpdateIndicator 更新指示器（整体替换）。
func (s *Store) UpdateIndicator(ind *domain.FaultIndicator) error {
	if ind.ID == "" {
		return domain.Invalidf("indicator id is required")
	}
	return s.mutate(func() error {
		if _, ok := s.indicators[ind.ID]; !ok {
			return domain.NotFoundf("indicator %s not found", ind.ID)
		}
		s.indicators[ind.ID] = ind
		return nil
	})
}

// GetIndicator 查询指示器。
func (s *Store) GetIndicator(id string) (*domain.FaultIndicator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ind, ok := s.indicators[id]
	if !ok {
		return nil, domain.NotFoundf("indicator %s not found", id)
	}
	return ind, nil
}

// ListIndicators 列出指示器，可按线路/区段过滤；结果按 ID 升序，保证确定性。
func (s *Store) ListIndicators(feederID, sectionID string) []*domain.FaultIndicator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.FaultIndicator, 0, len(s.indicators))
	for _, ind := range s.indicators {
		if feederID != "" && ind.FeederID != feederID {
			continue
		}
		if sectionID != "" && ind.SectionID != sectionID {
			continue
		}
		out = append(out, ind)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// DeleteIndicator 删除指示器。
func (s *Store) DeleteIndicator(id string) error {
	return s.mutate(func() error {
		if _, ok := s.indicators[id]; !ok {
			return domain.NotFoundf("indicator %s not found", id)
		}
		delete(s.indicators, id)
		return nil
	})
}

// CountTriggeredIndicators 统计翻牌指示器数量。
func (s *Store) CountTriggeredIndicators() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, ind := range s.indicators {
		if ind.IsTriggered() {
			n++
		}
	}
	return n
}
