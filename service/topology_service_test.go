package service

import (
	"errors"
	"testing"

	"example.com/grid-fault-locate-service/domain"
)

func TestTopology_AddSectionValidation(t *testing.T) {
	e := newTestEnv(t)
	feederID, _ := e.seedChain(t)

	// 重复端点拒绝
	_, err := e.topology.AddSection(feederID, SectionInput{Name: "dup", UpstreamSwitchID: swID(t, e, feederID, 1), DownstreamSwitchID: swID(t, e, feederID, 2), LengthKm: 1}, "t", "r")
	if !errors.Is(err, domain.ErrTopologyInvalid) {
		t.Fatalf("duplicate section should be topology invalid, got %v", err)
	}

	// 成环拒绝：SW-1 -> SW-3（中间已连通）
	if _, err := e.topology.AddSection(feederID, SectionInput{Name: "loop", UpstreamSwitchID: swID(t, e, feederID, 1), DownstreamSwitchID: swID(t, e, feederID, 3), LengthKm: 1}, "t", "r"); !errors.Is(err, domain.ErrTopologyInvalid) {
		t.Fatalf("cycle section should be rejected, got %v", err)
	}

	// 悬空拒绝：两端都是新开关
	extra, err := e.topology.AddSwitch(feederID, SwitchInput{Name: "孤立开关", SwitchType: domain.SwitchTypeSectionalizer}, "t", "r")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.topology.AddSection(feederID, SectionInput{Name: "dangling", UpstreamSwitchID: extra.ID, DownstreamSwitchID: swID(t, e, feederID, 4), LengthKm: 1}, "t", "r"); !errors.Is(err, domain.ErrTopologyInvalid) {
		t.Fatalf("dangling section should be rejected, got %v", err)
	}
}

func TestTopology_RemoveSectionDisconnect(t *testing.T) {
	e := newTestEnv(t)
	feederID, _ := e.seedChain(t)
	sws := e.store.ListSwitches(feederID)
	var outlet *domain.SwitchNode
	for _, sw := range sws {
		if sw.IsFeederOutlet() {
			outlet = sw
		}
	}
	if outlet == nil {
		t.Fatal("outlet missing")
	}
	// 删除区段2（SW-2->SW-3）会断开 SW-3/SW-4 → 拒绝
	sections := e.store.ListSections(feederID)
	mid := sections[1]
	if err := e.topology.RemoveSection(feederID, mid.ID, "t", "r"); !errors.Is(err, domain.ErrTopologyInvalid) {
		t.Fatalf("removing middle section should be rejected, got %v", err)
	}
	// 删除末端区段3 合法
	last := sections[2]
	if err := e.topology.RemoveSection(feederID, last.ID, "t", "r"); err != nil {
		t.Fatalf("removing tail section should succeed, got %v", err)
	}
}

func TestTopology_SwitchRules(t *testing.T) {
	e := newTestEnv(t)
	feederID, _ := e.seedChain(t)

	// 第二个出线开关拒绝
	if _, err := e.topology.AddSwitch(feederID, SwitchInput{Name: "出线2", SwitchType: domain.SwitchTypeFeederOutlet}, "t", "r"); err == nil {
		t.Fatal("second feeder outlet should be rejected")
	}

	// 删除被区段引用的开关拒绝
	sw := swID(t, e, feederID, 2)
	if err := e.topology.RemoveSwitch(feederID, sw, "t", "r"); err == nil {
		t.Fatal("removing switch referenced by section should be rejected")
	}

	// 删除出线开关拒绝
	outlet := swID(t, e, feederID, 1)
	if err := e.topology.RemoveSwitch(feederID, outlet, "t", "r"); err == nil {
		t.Fatal("removing feeder outlet should be rejected")
	}
}

func TestTopology_GetTopologyOrder(t *testing.T) {
	e := newTestEnv(t)
	feederID, _ := e.seedChain(t)
	tp, err := e.topology.GetTopology(feederID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tp.SectionOrder) != 3 {
		t.Fatalf("expected 3 ordered sections, got %v", tp.SectionOrder)
	}
	// 开关按 order 排序，出线开关在前
	if len(tp.Switches) != 4 || !tp.Switches[0].IsFeederOutlet() {
		t.Fatalf("switches not ordered with outlet first: %+v", tp.Switches)
	}
}

func swID(t *testing.T, e *testEnv, feederID string, order int) string {
	t.Helper()
	for _, sw := range e.store.ListSwitches(feederID) {
		if sw.Order == order {
			return sw.ID
		}
	}
	t.Fatalf("switch with order %d not found", order)
	return ""
}
