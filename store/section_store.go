package store

import (
	"sort"

	"example.com/grid-fault-locate-service/domain"
)

// CreateSection 新增线路区段。
func (s *Store) CreateSection(sec *domain.FeederSection) error {
	if sec.ID == "" {
		return domain.Invalidf("section id is required")
	}
	return s.mutate(func() error {
		if _, ok := s.sections[sec.ID]; ok {
			return domain.Conflictf("section %s already exists", sec.ID)
		}
		s.sections[sec.ID] = sec
		return nil
	})
}

// UpdateSection 更新区段（整体替换）。
func (s *Store) UpdateSection(sec *domain.FeederSection) error {
	if sec.ID == "" {
		return domain.Invalidf("section id is required")
	}
	return s.mutate(func() error {
		if _, ok := s.sections[sec.ID]; !ok {
			return domain.NotFoundf("section %s not found", sec.ID)
		}
		s.sections[sec.ID] = sec
		return nil
	})
}

// GetSection 查询区段。
func (s *Store) GetSection(id string) (*domain.FeederSection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sec, ok := s.sections[id]
	if !ok {
		return nil, domain.NotFoundf("section %s not found", id)
	}
	return sec, nil
}

// ListSections 列出区段，可选按线路过滤。
// 按 ID 升序返回，保证与创建顺序一致、消除 map 迭代的随机性，
// 使依赖下标的调用方（如拓扑删除连通性校验）稳定可复现。
func (s *Store) ListSections(feederID string) []*domain.FeederSection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.FeederSection, 0, len(s.sections))
	for _, sec := range s.sections {
		if feederID == "" || sec.FeederID == feederID {
			out = append(out, sec)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// DeleteSection 删除区段（调用方需先校验拓扑连通性）。
func (s *Store) DeleteSection(id string) error {
	return s.mutate(func() error {
		if _, ok := s.sections[id]; !ok {
			return domain.NotFoundf("section %s not found", id)
		}
		delete(s.sections, id)
		return nil
	})
}

// CountSectionsOfFeeder 统计线路的区段数量。
func (s *Store) CountSectionsOfFeeder(feederID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, sec := range s.sections {
		if sec.FeederID == feederID {
			n++
		}
	}
	return n
}
