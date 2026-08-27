package domain

import (
	"testing"
	"time"
)

func bug002Event(status FaultStatus) *FaultEvent {
	at := time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local)
	return &FaultEvent{ID: "FE-1", FeederID: "F1", FeederName: "测试线", Status: status, LocatedAt: at}
}

func TestBug002DomainRestoreRequiresIsolation(t *testing.T) {
	e := bug002Event(FaultRepairing)
	if err := e.ApplyRestore("op", "note", time.Now()); err == nil {
		t.Fatal("domain ApplyRestore 未隔离时不应成功")
	}
}

func TestBug002DomainArchiveFromLocated(t *testing.T) {
	e := bug002Event(FaultLocated)
	if err := e.ApplyArchive("op", time.Now()); err == nil {
		t.Fatal("domain ApplyArchive 未复电时不应成功")
	}
}

func TestBug002DomainArchiveFromRepairing(t *testing.T) {
	e := bug002Event(FaultRepairing)
	if err := e.ApplyArchive("op", time.Now()); err == nil {
		t.Fatal("domain ApplyArchive 抢修中不应成功")
	}
}
