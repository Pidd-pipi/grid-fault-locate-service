package domain

import (
	"testing"
	"time"
)

func mkSection(id, up, down string) *FeederSection {
	return NewFeederSection(id, "F-1", "sec-"+id, up, down, 1.0, time.Now())
}

func TestReachableFrom(t *testing.T) {
	sections := []*FeederSection{
		mkSection("S1", "SW-1", "SW-2"),
		mkSection("S2", "SW-2", "SW-3"),
	}
	reach := ReachableFrom(sections, "SW-1")
	if !reach["SW-1"] || !reach["SW-2"] || !reach["SW-3"] {
		t.Fatalf("all switches should be reachable from outlet: %v", reach)
	}
	if reach["SW-9"] {
		t.Fatal("SW-9 should not be reachable")
	}
}

func TestWouldCreateCycle(t *testing.T) {
	sections := []*FeederSection{
		mkSection("S1", "SW-1", "SW-2"),
		mkSection("S2", "SW-2", "SW-3"),
	}
	if !WouldCreateCycle(sections, "SW-1", "SW-3") {
		t.Fatal("SW-1..SW-3 already connected, adding edge must be flagged as cycle")
	}
	if !WouldCreateCycle(sections, "SW-3", "SW-1") {
		t.Fatal("reverse direction should also be flagged as cycle")
	}
	if WouldCreateCycle(sections, "SW-3", "SW-9") {
		t.Fatal("edge to a new node must not be flagged as cycle")
	}
}

func TestDownstreamOrder(t *testing.T) {
	sections := []*FeederSection{
		mkSection("S2", "SW-2", "SW-3"),
		mkSection("S1", "SW-1", "SW-2"),
		mkSection("S3", "SW-3", "SW-4"),
	}
	order := DownstreamOrder(sections, "SW-1")
	want := []string{"S1", "S2", "S3"}
	if len(order) != len(want) {
		t.Fatalf("want %d sections, got %d: %v", len(want), len(order), order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d]=%s want %s (full: %v)", i, order[i], want[i], order)
		}
	}
}

func TestNeighborSectionIDs(t *testing.T) {
	sections := []*FeederSection{
		mkSection("S1", "SW-1", "SW-2"),
		mkSection("S2", "SW-2", "SW-3"),
		mkSection("S3", "SW-3", "SW-4"),
	}
	nb := NeighborSectionIDs(sections, "S2")
	if len(nb) != 2 {
		t.Fatalf("S2 should have 2 neighbors, got %v", nb)
	}
	has := func(id string) bool {
		for _, n := range nb {
			if n == id {
				return true
			}
		}
		return false
	}
	if !has("S1") || !has("S3") {
		t.Fatalf("neighbors should be S1 and S3, got %v", nb)
	}
}

func TestFeederSectionValidate(t *testing.T) {
	sec := mkSection("S1", "SW-1", "SW-2")
	if err := sec.Validate(); err != nil {
		t.Fatalf("valid section should pass: %v", err)
	}
	bad := NewFeederSection("S2", "F-1", "bad", "SW-1", "SW-1", 1.0, time.Now())
	if err := bad.Validate(); err == nil {
		t.Fatal("section with same endpoints should fail")
	}
	neg := NewFeederSection("S3", "F-1", "neg", "SW-1", "SW-2", -1, time.Now())
	if err := neg.Validate(); err == nil {
		t.Fatal("section with negative length should fail")
	}
}
