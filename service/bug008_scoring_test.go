package service

import (
	"context"
	"testing"
	"time"

	"example.com/grid-fault-locate-service/config"
)

func bug008Sweeper(t *testing.T, interval time.Duration) (*testEnv, *LongOutageSweeper) {
	t.Helper()
	e := newTestEnv(t)
	cfg := config.Default()
	cfg.SweepInterval = interval
	cfg.LongOutageMinutes = 120
	return e, NewLongOutageSweeper(e.store, e.faults, e.audit, cfg)
}

func TestBug008SweeperStopsOnCancel(t *testing.T) {
	_, sweeper := bug008Sweeper(t, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sweeper.Start(ctx)
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("扫描器取消后未及时退出")
	}
}

func TestBug008ThresholdBoundary(t *testing.T) {
	e, sweeper := bug008Sweeper(t, time.Minute)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local)
	ev := e.seedLocatedFault(t, feederID, secs[0], base)
	flagged, err := sweeper.RunOnce(base.Add(2 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(flagged) != 1 || flagged[0] != ev.ID {
		t.Fatalf("定位满 2 小时应标记长时停电，实际 %v", flagged)
	}
}

func TestBug008OverviewLongOutageList(t *testing.T) {
	e, _ := bug008Sweeper(t, time.Minute)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local)
	longEv := e.seedLocatedFault(t, feederID, secs[0], base)
	longEv.LongOutage = true
	longEv.LongOutageSince = base
	_ = e.store.UpdateFault(longEv)
	e.seedLocatedFault(t, feederID, secs[1], base.Add(-10*time.Minute))
	ov := NewOverviewService(e.store, e.outages, e.faults)
	got := ov.GetOverview()
	if len(got.LongOutageFaults) != 1 || got.LongOutageFaults[0].ID != longEv.ID {
		t.Fatalf("长时停电列表应只含 LongOutage 故障，实际 %d 条", len(got.LongOutageFaults))
	}
	if got.LongOutageFaults == nil {
		t.Fatal("长时停电列表不应为 null")
	}
}

func TestBug008RecentFaultLimit(t *testing.T) {
	e, _ := bug008Sweeper(t, time.Minute)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local)
	for i := 0; i < 8; i++ {
		e.seedLocatedFault(t, feederID, secs[0], base.Add(time.Duration(i)*time.Minute))
	}
	ov := NewOverviewService(e.store, e.outages, e.faults)
	got := ov.GetOverview()
	if len(got.RecentFaults) != 5 {
		t.Fatalf("近期故障应最多 5 条，实际 %d", len(got.RecentFaults))
	}
}

func TestBug008RecentIncludesActive(t *testing.T) {
	e, _ := bug008Sweeper(t, time.Minute)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local)
	ev := e.seedLocatedFault(t, feederID, secs[0], base)
	ov := NewOverviewService(e.store, e.outages, e.faults)
	got := ov.GetOverview()
	if len(got.RecentFaults) < 1 || got.RecentFaults[0].ID != ev.ID {
		t.Fatalf("近期故障应包含活跃故障，实际 %d 条", len(got.RecentFaults))
	}
}
