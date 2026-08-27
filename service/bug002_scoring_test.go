package service

import (
	"testing"
	"time"
)

func bug002Setup(t *testing.T) (*testEnv, string, []string) {
	t.Helper()
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	return e, feederID, secs
}

func TestBug002RestoreRequiresIsolation(t *testing.T) {
	e, feederID, secs := bug002Setup(t)
	at := time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local)
	ev := e.seedLocatedFault(t, feederID, secs[0], at)
	if _, err := e.faults.StartRepair(ev.ID, "op", "repair", "req"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.faults.Restore(ev.ID, "op", "restore", "req"); err == nil {
		t.Fatal("未隔离直接复电不应成功")
	}
}

func TestBug002ArchiveRequiresRestored(t *testing.T) {
	e, feederID, secs := bug002Setup(t)
	at := time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local)
	ev := e.seedLocatedFault(t, feederID, secs[0], at)
	if _, err := e.faults.Archive(ev.ID, "op", "req"); err == nil {
		t.Fatal("未复电直接归档不应成功")
	}
}

func TestBug002IsolateValidatesFeeder(t *testing.T) {
	e, feederID, secs := bug002Setup(t)
	// 在另一条线路建一个区段，隔离时不得跨线路放行。
	f2, err := e.topology.CreateFeeder(FeederInput{Name: "另一线", Substation: "变电站B", VoltageLevel: "10kV", Status: "active"}, "op", "req")
	if err != nil {
		t.Fatal(err)
	}
	o, _ := e.topology.AddSwitch(f2.ID, SwitchInput{Name: "出线", SwitchType: "feeder_outlet"}, "op", "req")
	a, _ := e.topology.AddSwitch(f2.ID, SwitchInput{Name: "分段", SwitchType: "sectionalizer"}, "op", "req")
	sec2, err := e.topology.AddSection(f2.ID, SectionInput{Name: "区段X", UpstreamSwitchID: o.ID, DownstreamSwitchID: a.ID, LengthKm: 1.0}, "op", "req")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local)
	ev := e.seedLocatedFault(t, feederID, secs[0], at)
	if _, err := e.faults.Isolate(ev.ID, "op", sec2.ID, "iso", "req"); err == nil {
		t.Fatal("跨线路区段隔离不应成功")
	}
}
