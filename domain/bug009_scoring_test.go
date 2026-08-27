package domain

import (
	"errors"
	"testing"
)

func TestBug009InvalidSentinelPreserved(t *testing.T) {
	if !errors.Is(Invalidf("bad %s", "x"), ErrInvalid) {
		t.Fatal("Invalidf 应保留 ErrInvalid 哨兵错误链")
	}
}

func TestBug009NotFoundSentinelPreserved(t *testing.T) {
	if !errors.Is(NotFoundf("missing %s", "x"), ErrNotFound) {
		t.Fatal("NotFoundf 应保留 ErrNotFound 哨兵错误链")
	}
}

func TestBug009ConflictSentinelPreserved(t *testing.T) {
	if !errors.Is(Conflictf("dup %s", "x"), ErrConflict) {
		t.Fatal("Conflictf 应保留 ErrConflict 哨兵错误链")
	}
}

func TestBug009TopologySentinelPreserved(t *testing.T) {
	if !errors.Is(Topologyf("bad %s", "x"), ErrTopologyInvalid) {
		t.Fatal("Topologyf 应保留 ErrTopologyInvalid 哨兵错误链")
	}
}

func TestBug009StateSentinelPreserved(t *testing.T) {
	if !errors.Is(Statef("bad %s", "x"), ErrStateTransition) {
		t.Fatal("Statef 应保留 ErrStateTransition 哨兵错误链")
	}
}
