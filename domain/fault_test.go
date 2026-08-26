package domain

import (
	"testing"
	"time"
)

func TestFaultStatusMachine_ValidTransitions(t *testing.T) {
	now := time.Now()
	ev := NewFaultEvent("FE-1", "F-1", "线A", "SEC-1", []string{"SEC-1"}, nil, nil, "ev", "op", now)
	if ev.Status != FaultLocated {
		t.Fatalf("new event should be located, got %s", ev.Status)
	}

	// located -> repairing (repair)
	if err := ev.ApplyRepair("op1", "开始抢修", now.Add(time.Minute)); err != nil {
		t.Fatalf("repair should be allowed: %v", err)
	}
	if ev.Status != FaultRepairing {
		t.Fatalf("expected repairing, got %s", ev.Status)
	}

	// repairing -> restoring without isolation must fail
	if err := ev.ApplyRestore("op", "", now.Add(2*time.Minute)); err == nil {
		t.Fatal("restore without isolation should fail")
	}

	// repairing -> repairing (isolate)
	if err := ev.ApplyIsolate("SEC-1", "op2", "隔离", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("isolate should be allowed: %v", err)
	}
	if !ev.Isolated() {
		t.Fatal("event should be isolated")
	}

	// repairing -> restored
	if err := ev.ApplyRestore("op3", "复电", now.Add(4*time.Minute)); err != nil {
		t.Fatalf("restore after isolation should be allowed: %v", err)
	}
	if ev.Status != FaultRestored {
		t.Fatalf("expected restored, got %s", ev.Status)
	}
	if got := ev.OutageDurationMinutes(); got != 4 {
		t.Fatalf("expected 4 minutes outage, got %d", got)
	}

	// restored -> archived
	if err := ev.ApplyArchive("op4", now.Add(5*time.Minute)); err != nil {
		t.Fatalf("archive should be allowed: %v", err)
	}
	if ev.Status != FaultArchived {
		t.Fatalf("expected archived, got %s", ev.Status)
	}
}

func TestFaultStatusMachine_InvalidTransitions(t *testing.T) {
	now := time.Now()
	ev := NewFaultEvent("FE-2", "F-1", "线A", "SEC-1", []string{"SEC-1"}, nil, nil, "ev", "op", now)

	// located -> restore directly must fail (no isolation + invalid transition)
	if err := ev.ApplyRestore("op", "", now); err == nil {
		t.Fatal("restore from located should fail")
	}
	// located -> archive must fail
	if _, err := ev.ValidateAction(ActionArchive); err == nil {
		t.Fatal("archive from located should fail")
	}
	// restored event cannot repair again
	ev2 := NewFaultEvent("FE-3", "F-1", "线A", "SEC-1", []string{"SEC-1"}, nil, nil, "ev", "op", now)
	_ = ev2.ApplyIsolate("SEC-1", "op", "", now.Add(time.Minute))
	_ = ev2.ApplyRestore("op", "", now.Add(2*time.Minute))
	if err := ev2.ApplyRepair("op", "", now.Add(3*time.Minute)); err == nil {
		t.Fatal("repair from restored should fail")
	}
}

func TestFaultEvent_IsActive(t *testing.T) {
	now := time.Now()
	ev := NewFaultEvent("FE-4", "F-1", "线A", "SEC-1", []string{"SEC-1"}, nil, nil, "ev", "op", now)
	if !ev.Status.IsActive() {
		t.Fatal("located should be active")
	}
	if FaultRestored.IsActive() || FaultArchived.IsActive() {
		t.Fatal("restored/archived should not be active")
	}
}

func TestFaultEvent_MarkLongOutage(t *testing.T) {
	now := time.Now()
	ev := NewFaultEvent("FE-5", "F-1", "线A", "SEC-1", []string{"SEC-1"}, nil, nil, "ev", "op", now)
	ev.MarkLongOutage(now.Add(time.Hour))
	if !ev.LongOutage || ev.LongOutageSince.IsZero() {
		t.Fatal("long outage should be marked")
	}
	// 幂等：重复标记不改变时间
	first := ev.LongOutageSince
	ev.MarkLongOutage(now.Add(2 * time.Hour))
	if !ev.LongOutageSince.Equal(first) {
		t.Fatal("marking twice should keep first timestamp")
	}
}
