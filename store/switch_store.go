package store

import (
	"sort"

	"example.com/grid-fault-locate-service/domain"
)

// CreateSwitch 新增开关节点。
func (s *Store) CreateSwitch(sw *domain.SwitchNode) error {
	if sw.ID == "" {
		return domain.Invalidf("switch id is required")
	}
	return s.mutate(func() error {
		if _, ok := s.switches[sw.ID]; ok {
			return domain.Conflictf("switch %s already exists", sw.ID)
		}
		s.switches[sw.ID] = sw
		return nil
	})
}

// UpdateSwitch 更新开关节点（整体替换）。
func (s *Store) UpdateSwitch(sw *domain.SwitchNode) error {
	if sw.ID == "" {
		return domain.Invalidf("switch id is required")
	}
	return s.mutate(func() error {
		if _, ok := s.switches[sw.ID]; !ok {
			return domain.NotFoundf("switch %s not found", sw.ID)
		}
		s.switches[sw.ID] = sw
		return nil
	})
}

// GetSwitch 查询开关节点。
func (s *Store) GetSwitch(id string) (*domain.SwitchNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sw, ok := s.switches[id]
	if !ok {
		return nil, domain.NotFoundf("switch %s not found", id)
	}
	return sw, nil
}

// ListSwitches 列出开关节点，可选按线路过滤。
// 按 (Order, ID) 升序返回，消除 map 迭代随机性，保证出线开关在前、
// 展示与下游顺序一致。
func (s *Store) ListSwitches(feederID string) []*domain.SwitchNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.SwitchNode, 0, len(s.switches))
	for _, sw := range s.switches {
		if feederID == "" || sw.FeederID == feederID {
			out = append(out, sw)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// DeleteSwitch 删除开关节点（调用方需先校验其非区段端点）。
func (s *Store) DeleteSwitch(id string) error {
	return s.mutate(func() error {
		if _, ok := s.switches[id]; !ok {
			return domain.NotFoundf("switch %s not found", id)
		}
		delete(s.switches, id)
		return nil
	})
}

// CountSwitchesOfFeeder 统计线路的开关数量。
func (s *Store) CountSwitchesOfFeeder(feederID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, sw := range s.switches {
		if sw.FeederID == feederID {
			n++
		}
	}
	return n
}
