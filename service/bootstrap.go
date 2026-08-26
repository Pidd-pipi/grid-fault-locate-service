package service

import (
	"time"

	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/store"
)

// Bootstrap 首次启动时的演示数据初始化：空库时写入两条线路及其
// 开关/区段/指示器（全部复位），便于页面展示与后续故障演练。
type Bootstrap struct {
	store    *store.Store
	topology *TopologyService
	signals  *SignalService
}

// NewBootstrap 构造引导器。
func NewBootstrap(st *store.Store, topology *TopologyService, signals *SignalService) *Bootstrap {
	return &Bootstrap{store: st, topology: topology, signals: signals}
}

// SeedIfEmpty 仅当仓储为空（无任何线路）时写入演示数据。
func (b *Bootstrap) SeedIfEmpty() error {
	if len(b.store.ListFeeders()) > 0 {
		return nil
	}
	now := time.Now()
	// 线路一：城东 10kV 线。
	f1 := domain.NewFeeder(b.store.NextID("F"), "城东10kV线", "城东变电站", "10kV", domain.FeederActive, now)
	if err := b.store.CreateFeeder(f1); err != nil {
		return err
	}
	sw1 := b.mustAddSwitch(f1.ID, "出线开关", domain.SwitchTypeFeederOutlet, domain.SwitchClosed, 1, now)
	sw2 := b.mustAddSwitch(f1.ID, "分段开关A", domain.SwitchTypeSectionalizer, domain.SwitchClosed, 2, now)
	sw3 := b.mustAddSwitch(f1.ID, "分段开关B", domain.SwitchTypeSectionalizer, domain.SwitchClosed, 3, now)
	sw4 := b.mustAddSwitch(f1.ID, "联络开关1", domain.SwitchTypeTie, domain.SwitchOpen, 4, now)
	sec1 := b.mustAddSection(f1.ID, "区段1", sw1.ID, sw2.ID, 1.2, now)
	sec2 := b.mustAddSection(f1.ID, "区段2", sw2.ID, sw3.ID, 0.9, now)
	sec3 := b.mustAddSection(f1.ID, "区段3", sw3.ID, sw4.ID, 1.5, now)
	_ = b.mustAddIndicator(sec1, "FI-101", 0.4, now)
	_ = b.mustAddIndicator(sec2, "FI-102", 0.5, now)
	_ = b.mustAddIndicator(sec3, "FI-103", 0.8, now)

	// 线路二：城西 10kV 线（带一条分支）。
	f2 := domain.NewFeeder(b.store.NextID("F"), "城西10kV线", "城西变电站", "10kV", domain.FeederActive, now)
	if err := b.store.CreateFeeder(f2); err != nil {
		return err
	}
	o2 := b.mustAddSwitch(f2.ID, "出线开关2", domain.SwitchTypeFeederOutlet, domain.SwitchClosed, 1, now)
	s2a := b.mustAddSwitch(f2.ID, "分段开关C", domain.SwitchTypeSectionalizer, domain.SwitchClosed, 2, now)
	s2b := b.mustAddSwitch(f2.ID, "分段开关D", domain.SwitchTypeSectionalizer, domain.SwitchClosed, 3, now)
	_ = b.mustAddSection(f2.ID, "区段A", o2.ID, s2a.ID, 2.0, now)
	secB := b.mustAddSection(f2.ID, "区段B", s2a.ID, s2b.ID, 1.1, now)
	_ = b.mustAddIndicator(secB, "FI-201", 0.3, now)
	return nil
}

func (b *Bootstrap) mustAddSwitch(feederID, name string, swType domain.SwitchType, status domain.SwitchStatus, order int, now time.Time) *domain.SwitchNode {
	sw := domain.NewSwitchNode(b.store.NextID("SW"), feederID, name, swType, status, order, now)
	if err := b.store.CreateSwitch(sw); err != nil {
		panic(err)
	}
	return sw
}

func (b *Bootstrap) mustAddSection(feederID, name, up, down string, length float64, now time.Time) *domain.FeederSection {
	sec := domain.NewFeederSection(b.store.NextID("SEC"), feederID, name, up, down, length, now)
	if err := b.store.CreateSection(sec); err != nil {
		panic(err)
	}
	return sec
}

func (b *Bootstrap) mustAddIndicator(sec *domain.FeederSection, name string, pos float64, now time.Time) *domain.FaultIndicator {
	ind := domain.NewFaultIndicator(b.store.NextID("FI"), sec.FeederID, sec.ID, name, pos, now)
	if err := b.store.CreateIndicator(ind); err != nil {
		panic(err)
	}
	return ind
}
