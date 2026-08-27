package store

import (
	"testing"

	"example.com/grid-fault-locate-service/domain"
)

func TestBug004ListSectionsReturnsIndependentCopy(t *testing.T) {
	st, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	_ = st.CreateSection(&domain.FeederSection{ID: "SEC-A1", FeederID: "F-A", Name: "a1", UpstreamSwitchID: "u1", DownstreamSwitchID: "d1", LengthKm: 1})
	_ = st.CreateSection(&domain.FeederSection{ID: "SEC-A2", FeederID: "F-A", Name: "a2", UpstreamSwitchID: "u2", DownstreamSwitchID: "d2", LengthKm: 1})
	first := st.ListSections("F-A")
	if len(first) != 2 {
		t.Fatalf("线路 F-A 应有两个区段，实际 %d", len(first))
	}
	_ = st.CreateSection(&domain.FeederSection{ID: "SEC-B1", FeederID: "F-B", Name: "b1", UpstreamSwitchID: "x1", DownstreamSwitchID: "y1", LengthKm: 1})
	_ = st.CreateSection(&domain.FeederSection{ID: "SEC-B2", FeederID: "F-B", Name: "b2", UpstreamSwitchID: "x2", DownstreamSwitchID: "y2", LengthKm: 1})
	_ = st.ListSections("F-B")
	// first 不应被第二次查询污染
	ids := map[string]bool{}
	for _, sec := range first {
		ids[sec.ID] = true
	}
	if !ids["SEC-A1"] || !ids["SEC-A2"] {
		t.Fatalf("第一次查询结果被第二次查询污染: %v", first)
	}
	if len(first) != 2 {
		t.Fatalf("第一次查询结果长度被污染: %d", len(first))
	}
}
