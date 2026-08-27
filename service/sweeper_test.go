package service

import (
	"testing"
	"time"

	"example.com/grid-fault-locate-service/config"
	"example.com/grid-fault-locate-service/domain"
)

// seedLocatedFault 直接构造指定定位时间的故障事件（绕过 time.Now）。
func (e *testEnv) seedLocatedFault(t *testing.T, feederID, sectionID string, locatedAt time.Time) *domain.FaultEvent {
	t.Helper()
	ev := domain.NewFaultEvent(e.store.NextID("FE"), feederID, "测试线", sectionID, []string{sectionID}, nil, nil, "ev", "op", locatedAt)
	if err := e.store.CreateFault(ev); err != nil {
		t.Fatal(err)
	}
	return ev
}

func TestSweeper_FlagsLongOutage(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local)
	ev := e.seedLocatedFault(t, feederID, secs[0], base)

	// 定位后 1 小时：未超阈值，不标记。
	flagged, err := e.sweeper.RunOnce(base.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(flagged) != 0 {
		t.Fatalf("1 hour should not flag, got %v", flagged)
	}
	got, _ := e.store.GetFault(ev.ID)
	if got.LongOutage {
		t.Fatal("should not be long outage yet")
	}

	// 定位后 3 小时：超阈值，标记。
	flagged, err = e.sweeper.RunOnce(base.Add(3 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(flagged) != 1 || flagged[0] != ev.ID {
		t.Fatalf("expected %s flagged, got %v", ev.ID, flagged)
	}
	got, _ = e.store.GetFault(ev.ID)
	if !got.LongOutage || got.LongOutageSince.IsZero() {
		t.Fatal("event should be marked long outage")
	}

	// 已复电事件不再被标记。
	_, _ = e.faults.Isolate(ev.ID, "op", secs[0], "iso", "req")
	_, _ = e.faults.Restore(ev.ID, "op", "restore", "req")
	flagged, err = e.sweeper.RunOnce(base.Add(5 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(flagged) != 0 {
		t.Fatalf("restored event should not be flagged again, got %v", flagged)
	}
}

func TestSweeper_RespectsConfiguredThreshold(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local)
	ev := e.seedLocatedFault(t, feederID, secs[0], base)

	// 自定义阈值 30 分钟。
	cfg := config.Default()
	cfg.LongOutageMinutes = 30
	sweeper := NewLongOutageSweeper(e.store, e.faults, e.audit, cfg)
	flagged, err := sweeper.RunOnce(base.Add(45 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(flagged) != 1 || flagged[0] != ev.ID {
		t.Fatalf("30min threshold should flag at 45min, got %v", flagged)
	}
}
