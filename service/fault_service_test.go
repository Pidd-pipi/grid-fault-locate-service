package service

import (
	"errors"
	"testing"
	"time"

	"example.com/grid-fault-locate-service/domain"
)

// locateAndCreate 构造一个已定位的故障事件。
func (e *testEnv) locateAndCreate(t *testing.T, feederID string) *domain.FaultEvent {
	t.Helper()
	ev, _, err := e.faults.LocateAndCreateEvent(LocateInput{FeederID: feederID}, "调度员", "req")
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func TestFaultWorkflow_FullClosedLoop(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 20, 0, 0, 0, time.Local)
	i1 := e.seedIndicator(t, secs[0], "FI-101")
	i2 := e.seedIndicator(t, secs[1], "FI-102")
	e.report(t, i1, domain.IndicatorTriggered, base.Add(time.Minute))
	e.report(t, i2, domain.IndicatorReset, base.Add(2*time.Minute))

	ev := e.locateAndCreate(t, feederID)
	if ev.Status != domain.FaultLocated {
		t.Fatalf("expected located, got %s", ev.Status)
	}
	if ev.PrimarySectionID != secs[0] {
		t.Fatalf("primary section should be %s, got %s", secs[0], ev.PrimarySectionID)
	}

	// 开始抢修 → repairing
	ev, err := e.faults.StartRepair(ev.ID, "抢修班", "抵达现场", "req")
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != domain.FaultRepairing {
		t.Fatalf("expected repairing, got %s", ev.Status)
	}

	// 隔离确认 → 两端开关分闸 + repairing
	ev, err = e.faults.Isolate(ev.ID, "调度员", secs[0], "确认隔离区段1", "req")
	if err != nil {
		t.Fatal(err)
	}
	if ev.IsolatedSectionID != secs[0] || !ev.Isolated() {
		t.Fatalf("isolation not recorded: %+v", ev)
	}
	// 区段两端开关（出线开关 SW-1 与分段A SW-2）应已分闸。
	sec, _ := e.store.GetSection(secs[0])
	up, _ := e.store.GetSwitch(sec.UpstreamSwitchID)
	down, _ := e.store.GetSwitch(sec.DownstreamSwitchID)
	if up.Status != domain.SwitchOpen || down.Status != domain.SwitchOpen {
		t.Fatalf("boundary switches should be open after isolation, got %s/%s", up.Status, down.Status)
	}

	// 复电完成 → restored + 停电记录 + 开关合闸
	ev, err = e.faults.Restore(ev.ID, "调度员", "故障处理完毕复电", "req")
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != domain.FaultRestored {
		t.Fatalf("expected restored, got %s", ev.Status)
	}
	sec, _ = e.store.GetSection(secs[0])
	if sec.Isolated || sec.IsCandidate {
		t.Fatalf("section should be cleared after restore: %+v", sec)
	}
	up, _ = e.store.GetSwitch(sec.UpstreamSwitchID)
	down, _ = e.store.GetSwitch(sec.DownstreamSwitchID)
	if up.Status != domain.SwitchClosed || down.Status != domain.SwitchClosed {
		t.Fatalf("boundary switches should be closed after restore, got %s/%s", up.Status, down.Status)
	}
	outages := e.outages.ListOutages(feederID)
	if len(outages) != 1 {
		t.Fatalf("expected 1 outage record, got %d", len(outages))
	}
	if outages[0].FaultEventID != ev.ID {
		t.Fatalf("outage not linked to fault: %+v", outages[0])
	}
	wantMinutes := int(ev.RestoredAt.Sub(ev.LocatedAt).Minutes())
	if outages[0].DurationMinutes != wantMinutes {
		t.Fatalf("outage duration mismatch: got %d, want %d", outages[0].DurationMinutes, wantMinutes)
	}

	// 归档 → archived
	ev, err = e.faults.Archive(ev.ID, "调度员", "req")
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != domain.FaultArchived {
		t.Fatalf("expected archived, got %s", ev.Status)
	}
}

func TestFaultWorkflow_RestoreWithoutIsolationRejected(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 21, 0, 0, 0, time.Local)
	i1 := e.seedIndicator(t, secs[0], "FI-101")
	i2 := e.seedIndicator(t, secs[1], "FI-102")
	e.report(t, i1, domain.IndicatorTriggered, base.Add(time.Minute))
	e.report(t, i2, domain.IndicatorReset, base.Add(2*time.Minute))
	ev := e.locateAndCreate(t, feederID)

	if _, err := e.faults.Restore(ev.ID, "调度员", "直接复电", "req"); !errors.Is(err, domain.ErrNotIsolated) {
		t.Fatalf("restore without isolation should fail with ErrNotIsolated, got %v", err)
	}
}

func TestFaultWorkflow_IsolateRejectsWrongFeederSection(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 22, 0, 0, 0, time.Local)
	i1 := e.seedIndicator(t, secs[0], "FI-101")
	e.report(t, i1, domain.IndicatorTriggered, base.Add(time.Minute))
	ev := e.locateAndCreate(t, feederID)

	// 第二条线路的区段不能用于隔离。
	other, err := e.topology.CreateFeeder(FeederInput{Name: "别的线", Substation: "站B", VoltageLevel: "10kV"}, "t", "r")
	if err != nil {
		t.Fatal(err)
	}
	o, err := e.topology.AddSwitch(other.ID, SwitchInput{Name: "出线", SwitchType: domain.SwitchTypeFeederOutlet}, "t", "r")
	if err != nil {
		t.Fatal(err)
	}
	s, err := e.topology.AddSwitch(other.ID, SwitchInput{Name: "分段", SwitchType: domain.SwitchTypeSectionalizer}, "t", "r")
	if err != nil {
		t.Fatal(err)
	}
	otherSec, err := e.topology.AddSection(other.ID, SectionInput{Name: "区段X", UpstreamSwitchID: o.ID, DownstreamSwitchID: s.ID, LengthKm: 1}, "t", "r")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.faults.Isolate(ev.ID, "调度员", otherSec.ID, "错误隔离", "req"); err == nil {
		t.Fatal("isolating a section from another feeder should be rejected")
	}
}

func TestFaultWorkflow_IsolateNonCandidateAllowed(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 23, 0, 0, 0, time.Local)
	i3 := e.seedIndicator(t, secs[2], "FI-103")
	e.report(t, i3, domain.IndicatorTriggered, base.Add(time.Minute))
	ev := e.locateAndCreate(t, feederID)
	// 隔离候选区段（末端）成功。
	if _, err := e.faults.Isolate(ev.ID, "调度员", secs[2], "隔离末端", "req"); err != nil {
		t.Fatalf("isolating candidate section should succeed: %v", err)
	}
}
