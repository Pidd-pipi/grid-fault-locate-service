package domain

import (
	"testing"
)

func bug004Sections() []*FeederSection {
	return []*FeederSection{
		{ID: "S1", UpstreamSwitchID: "A", DownstreamSwitchID: "B"},
		{ID: "S2", UpstreamSwitchID: "B", DownstreamSwitchID: "C"},
		{ID: "S3", UpstreamSwitchID: "C", DownstreamSwitchID: "D"},
	}
}

func TestBug004DownstreamOrderReturnsIndependentCopy(t *testing.T) {
	sections := bug004Sections()
	first := DownstreamOrder(sections, "A")
	if len(first) != 3 {
		t.Fatalf("应返回 3 个区段，实际 %d", len(first))
	}
	// 用另一条更短的链覆盖共享底层
	other := []*FeederSection{{ID: "X1", UpstreamSwitchID: "M", DownstreamSwitchID: "N"}}
	_ = DownstreamOrder(other, "M")
	if len(first) != 3 || first[0] != "S1" || first[1] != "S2" || first[2] != "S3" {
		t.Fatalf("第一次结果被第二次调用污染: %v", first)
	}
}

func TestBug004NeighborSectionIDsReturnsIndependentCopy(t *testing.T) {
	sections := bug004Sections()
	first := NeighborSectionIDs(sections, "S2")
	if len(first) != 2 {
		t.Fatalf("S2 应有两个邻居，实际 %d", len(first))
	}
	other := []*FeederSection{{ID: "X1", UpstreamSwitchID: "M", DownstreamSwitchID: "N"}, {ID: "X2", UpstreamSwitchID: "N", DownstreamSwitchID: "O"}}
	_ = NeighborSectionIDs(other, "X1")
	if len(first) != 2 {
		t.Fatalf("第一次邻居结果被污染，长度 %d", len(first))
	}
	for _, id := range first {
		if id != "S1" && id != "S3" {
			t.Fatalf("第一次邻居结果被污染: %v", first)
		}
	}
}

func TestBug004ReachableFromIndependentState(t *testing.T) {
	sections := bug004Sections()
	first := ReachableFrom(sections, "A")
	if first["D"] != true {
		t.Fatalf("A 应可达 D")
	}
	// 另一条独立链，第二次调用不得复用第一次的 seen
	other := []*FeederSection{{ID: "X1", UpstreamSwitchID: "M", DownstreamSwitchID: "N"}}
	second := ReachableFrom(other, "M")
	if second["D"] {
		t.Fatalf("第二次调用复用了第一次的 seen 状态，错误包含 D")
	}
	if !second["M"] || !second["N"] {
		t.Fatalf("第二次调用结果异常: %v", second)
	}
}
