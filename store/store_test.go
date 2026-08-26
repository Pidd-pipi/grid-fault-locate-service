package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/grid-fault-locate-service/domain"
)

func TestStoreCRUD(t *testing.T) {
	st, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	f := domain.NewFeeder(st.NextID("F"), "测试线", "变电站A", "10kV", domain.FeederActive, now)
	if err := st.CreateFeeder(f); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetFeeder(f.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "测试线" {
		t.Fatalf("unexpected name %s", got.Name)
	}
	if _, err := st.GetFeeder("F-NOPE"); err == nil {
		t.Fatal("missing feeder should error")
	}
	if len(st.ListFeeders()) != 1 {
		t.Fatal("expected 1 feeder")
	}
	// 重复创建冲突
	if err := st.CreateFeeder(domain.NewFeeder(f.ID, "x", "y", "10kV", domain.FeederActive, now)); err == nil {
		t.Fatal("duplicate create should conflict")
	}
	if err := st.DeleteFeeder(f.ID); err != nil {
		t.Fatal(err)
	}
	if len(st.ListFeeders()) != 0 {
		t.Fatal("feeder should be deleted")
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	st, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	f := domain.NewFeeder(st.NextID("F"), "持久化线", "变电站B", "20kV", domain.FeederActive, now)
	if err := st.CreateFeeder(f); err != nil {
		t.Fatal(err)
	}
	sw := domain.NewSwitchNode(st.NextID("SW"), f.ID, "出线开关", domain.SwitchTypeFeederOutlet, domain.SwitchClosed, 1, now)
	if err := st.CreateSwitch(sw); err != nil {
		t.Fatal(err)
	}
	sec := domain.NewFeederSection(st.NextID("SEC"), f.ID, "区段1", sw.ID, "SW-999", 1.2, now)
	if err := st.CreateSection(sec); err != nil {
		t.Fatal(err)
	}

	// 重新加载
	st2, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	f2, err := st2.GetFeeder(f.ID)
	if err != nil {
		t.Fatal(err)
	}
	if f2.Name != "持久化线" || f2.VoltageLevel != "20kV" {
		t.Fatalf("roundtrip mismatch: %+v", f2)
	}
	if len(st2.ListSwitches(f.ID)) != 1 || len(st2.ListSections(f.ID)) != 1 {
		t.Fatalf("switches/sections not persisted")
	}
	// 恢复序列号
	sw2 := domain.NewSwitchNode(st2.NextID("SW"), f.ID, "新开关", domain.SwitchTypeSectionalizer, domain.SwitchClosed, 2, now)
	if err := st2.CreateSwitch(sw2); err != nil {
		t.Fatal(err)
	}
	if sw2.ID != "SW-002" {
		t.Fatalf("sequence not restored, got %s", sw2.ID)
	}
}

func TestAuditStore(t *testing.T) {
	st, _ := New("")
	now := time.Now()
	for i := 0; i < 5; i++ {
		entry := domain.NewAuditEntry(st.NextID("AU"), "req-1", domain.AuditFaultLocate, "fault", "FE-1", "op", "detail", now)
		if err := st.AppendAudit(entry); err != nil {
			t.Fatal(err)
		}
	}
	list := st.ListAudits(100)
	if len(list) != 5 {
		t.Fatalf("expected 5 audits, got %d", len(list))
	}
	// 新→旧
	if list[0].ID != "AU-005" {
		t.Fatalf("newest first expected AU-005, got %s", list[0].ID)
	}
	if st.CountAudits() != 5 {
		t.Fatal("count mismatch")
	}
	// 删除后的文件清理
	_ = os.RemoveAll(filepath.Dir(st.PersistPath()))
}
