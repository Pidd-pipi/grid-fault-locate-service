package service

import (
	"testing"
	"time"

	"example.com/grid-fault-locate-service/config"
	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/store"
)

// testEnv 组装一套完整服务依赖（内存仓储，无持久化）。
type testEnv struct {
	store    *store.Store
	topology *TopologyService
	signals  *SignalService
	locate   *LocateService
	faults   *FaultService
	outages  *OutageService
	audit    *AuditService
	sweeper  *LongOutageSweeper
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	st, err := store.New("")
	if err != nil {
		t.Fatal(err)
	}
	audit := NewAuditService(st)
	topology := NewTopologyService(st, audit)
	signals := NewSignalService(st, audit)
	locate := NewLocateService(st, topology, audit)
	outages := NewOutageService(st, audit)
	faults := NewFaultService(st, locate, outages, audit)
	cfg := config.Default()
	sweeper := NewLongOutageSweeper(st, faults, audit, cfg)
	return &testEnv{
		store: st, topology: topology, signals: signals, locate: locate,
		faults: faults, outages: outages, audit: audit, sweeper: sweeper,
	}
}

// seedChain 构造一条 出线开关-分段A-分段B-联络 的四节点链式线路。
// 返回 feederID 与各区段 ID（按下游顺序）。
func (e *testEnv) seedChain(t *testing.T) (string, []string) {
	t.Helper()
	f, err := e.topology.CreateFeeder(FeederInput{
		Name: "测试线", Substation: "变电站A", VoltageLevel: "10kV", Status: domain.FeederActive,
	}, "tester", "req")
	if err != nil {
		t.Fatal(err)
	}
	outlet, err := e.topology.AddSwitch(f.ID, SwitchInput{Name: "出线开关", SwitchType: domain.SwitchTypeFeederOutlet}, "tester", "req")
	if err != nil {
		t.Fatal(err)
	}
	a, err := e.topology.AddSwitch(f.ID, SwitchInput{Name: "分段A", SwitchType: domain.SwitchTypeSectionalizer}, "tester", "req")
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.topology.AddSwitch(f.ID, SwitchInput{Name: "分段B", SwitchType: domain.SwitchTypeSectionalizer}, "tester", "req")
	if err != nil {
		t.Fatal(err)
	}
	tie, err := e.topology.AddSwitch(f.ID, SwitchInput{Name: "联络", SwitchType: domain.SwitchTypeTie}, "tester", "req")
	if err != nil {
		t.Fatal(err)
	}
	s1, err := e.topology.AddSection(f.ID, SectionInput{Name: "区段1", UpstreamSwitchID: outlet.ID, DownstreamSwitchID: a.ID, LengthKm: 1.2}, "tester", "req")
	if err != nil {
		t.Fatal(err)
	}
	s2, err := e.topology.AddSection(f.ID, SectionInput{Name: "区段2", UpstreamSwitchID: a.ID, DownstreamSwitchID: b.ID, LengthKm: 0.9}, "tester", "req")
	if err != nil {
		t.Fatal(err)
	}
	s3, err := e.topology.AddSection(f.ID, SectionInput{Name: "区段3", UpstreamSwitchID: b.ID, DownstreamSwitchID: tie.ID, LengthKm: 1.5}, "tester", "req")
	if err != nil {
		t.Fatal(err)
	}
	return f.ID, []string{s1.ID, s2.ID, s3.ID}
}

// seedIndicator 在指定区段上新增指示器并返回其 ID。
func (e *testEnv) seedIndicator(t *testing.T, sectionID string, name string) string {
	t.Helper()
	ind, err := e.signals.CreateIndicator(IndicatorInput{Name: name, SectionID: sectionID, Position: 0.5}, "tester", "req")
	if err != nil {
		t.Fatal(err)
	}
	return ind.ID
}

// report 上报信号（时间递增，保证确定性排序）。
func (e *testEnv) report(t *testing.T, indicatorID string, status domain.IndicatorStatus, at time.Time) {
	t.Helper()
	if _, err := e.signals.ReportSignal(indicatorID, status, at, "tester", "req"); err != nil {
		t.Fatal(err)
	}
}
