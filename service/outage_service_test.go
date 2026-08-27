package service

import (
	"testing"
	"time"

	"example.com/grid-fault-locate-service/domain"
)

func TestOutageService_Summary(t *testing.T) {
	e := newTestEnv(t)
	st := e.store
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.Local)

	// 手工构造两个已复电事件，生成停电记录。
	ev1 := domain.NewFaultEvent(st.NextID("FE"), "F-1", "线A", "SEC-1", []string{"SEC-1"}, nil, nil, "ev", "op", now)
	ev1.ApplyIsolate("SEC-1", "op", "", now.Add(10*time.Minute))
	ev1.ApplyRestore("op", "", now.Add(30*time.Minute)) // 30 分钟
	if _, err := e.outages.CreateFromRestore(ev1); err != nil {
		t.Fatal(err)
	}
	ev2 := domain.NewFaultEvent(st.NextID("FE"), "F-2", "线B", "SEC-9", []string{"SEC-9"}, nil, nil, "ev", "op", now.Add(-24*time.Hour))
	ev2.ApplyIsolate("SEC-9", "op", "", now.Add(-24*time.Hour+time.Minute))
	ev2.ApplyRestore("op", "", now.Add(-24*time.Hour+3*time.Hour)) // 180 分钟 → 长时停电
	if _, err := e.outages.CreateFromRestore(ev2); err != nil {
		t.Fatal(err)
	}

	summary := e.outages.Summary()
	if summary.TotalRecords != 2 {
		t.Fatalf("expected 2 records, got %d", summary.TotalRecords)
	}
	if summary.TotalMinutes != 210 {
		t.Fatalf("expected 210 total minutes, got %d", summary.TotalMinutes)
	}
	if summary.MaxMinutes != 180 {
		t.Fatalf("expected max 180, got %d", summary.MaxMinutes)
	}
	if summary.LongOutageCount != 1 {
		t.Fatalf("expected 1 long outage, got %d", summary.LongOutageCount)
	}
	if summary.AvgMinutes != 105 {
		t.Fatalf("expected avg 105, got %v", summary.AvgMinutes)
	}
	if len(summary.ByFeeder) != 2 {
		t.Fatalf("expected 2 feeder stats, got %d", len(summary.ByFeeder))
	}
}

func TestOutageRecord_DurationAndLongOutage(t *testing.T) {
	now := time.Now()
	ev := domain.NewFaultEvent("FE-9", "F-1", "线A", "SEC-1", []string{"SEC-1"}, nil, nil, "ev", "op", now)
	ev.ApplyIsolate("SEC-1", "op", "", now.Add(time.Minute))
	ev.ApplyRestore("op", "", now.Add(3*time.Hour))
	rec := domain.NewOutageRecord("OR-1", ev, now.Add(3*time.Hour))
	if rec.DurationMinutes != 180 {
		t.Fatalf("expected 180 minutes, got %d", rec.DurationMinutes)
	}
	if !rec.LongOutage {
		t.Fatal("180 minutes should be long outage")
	}
}
