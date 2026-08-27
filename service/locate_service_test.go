package service

import (
	"errors"
	"testing"
	"time"

	"example.com/grid-fault-locate-service/domain"
)

func TestLocate_BetweenLastTriggeredAndFirstReset(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 10, 0, 0, 0, time.Local)
	i1 := e.seedIndicator(t, secs[0], "FI-101")
	i2 := e.seedIndicator(t, secs[1], "FI-102")
	i3 := e.seedIndicator(t, secs[2], "FI-103")
	e.report(t, i1, domain.IndicatorTriggered, base.Add(time.Minute))
	e.report(t, i2, domain.IndicatorTriggered, base.Add(2*time.Minute))
	e.report(t, i3, domain.IndicatorReset, base.Add(3*time.Minute))

	res, err := e.locate.Locate(LocateInput{FeederID: feederID})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].SectionID != secs[1] {
		t.Fatalf("expected candidate %s, got %+v", secs[1], res.Candidates)
	}
	if res.PrimarySectionID != secs[1] {
		t.Fatalf("primary should be %s, got %s", secs[1], res.PrimarySectionID)
	}
	if len(res.Triggered) != 2 {
		t.Fatalf("expected 2 triggered signals, got %d", len(res.Triggered))
	}
	if len(res.Suspicious) != 0 {
		t.Fatalf("expected no suspicious, got %+v", res.Suspicious)
	}
}

func TestLocate_BoundaryFirstSection(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 11, 0, 0, 0, time.Local)
	i1 := e.seedIndicator(t, secs[0], "FI-101")
	i2 := e.seedIndicator(t, secs[1], "FI-102")
	e.report(t, i1, domain.IndicatorTriggered, base.Add(time.Minute))
	e.report(t, i2, domain.IndicatorReset, base.Add(2*time.Minute))

	res, err := e.locate.Locate(LocateInput{FeederID: feederID})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].SectionID != secs[0] {
		t.Fatalf("expected candidate %s, got %+v", secs[0], res.Candidates)
	}
}

func TestLocate_TailFaultNoResetDownstream(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 12, 0, 0, 0, time.Local)
	i3 := e.seedIndicator(t, secs[2], "FI-103")
	e.report(t, i3, domain.IndicatorTriggered, base.Add(time.Minute))

	res, err := e.locate.Locate(LocateInput{FeederID: feederID})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].SectionID != secs[2] {
		t.Fatalf("tail fault should locate at %s, got %+v", secs[2], res.Candidates)
	}
}

func TestLocate_MultiCandidateBetween(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 13, 0, 0, 0, time.Local)
	i1 := e.seedIndicator(t, secs[0], "FI-101")
	i3 := e.seedIndicator(t, secs[2], "FI-103")
	e.report(t, i1, domain.IndicatorTriggered, base.Add(time.Minute))
	e.report(t, i3, domain.IndicatorReset, base.Add(2*time.Minute))

	res, err := e.locate.Locate(LocateInput{FeederID: feederID})
	if err != nil {
		t.Fatal(err)
	}
	// 候选应为 sec1 与 sec2（两者之间无指示器），按信号时间排序。
	if len(res.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %+v", res.Candidates)
	}
	if res.Candidates[0].SectionID != secs[0] || res.Candidates[1].SectionID != secs[1] {
		t.Fatalf("unexpected candidate order: %+v", res.Candidates)
	}
	if res.PrimarySectionID != secs[0] {
		t.Fatalf("primary should be %s (latest signal), got %s", secs[0], res.PrimarySectionID)
	}
}

func TestLocate_WithinSectionConflict(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 14, 0, 0, 0, time.Local)
	i1 := e.seedIndicator(t, secs[0], "FI-101")
	i2 := e.seedIndicator(t, secs[1], "FI-102")
	i2b := e.seedIndicator(t, secs[1], "FI-102B")
	e.report(t, i1, domain.IndicatorTriggered, base.Add(time.Minute))
	e.report(t, i2, domain.IndicatorTriggered, base.Add(2*time.Minute))
	e.report(t, i2b, domain.IndicatorReset, base.Add(3*time.Minute))

	res, err := e.locate.Locate(LocateInput{FeederID: feederID})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].SectionID != secs[1] {
		t.Fatalf("same-section conflict should locate at %s, got %+v", secs[1], res.Candidates)
	}
}

func TestLocate_IsolatedTriggeredFlaggedSuspicious(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 15, 0, 0, 0, time.Local)
	i1 := e.seedIndicator(t, secs[0], "FI-101") // 孤立翻牌
	i2 := e.seedIndicator(t, secs[1], "FI-102") // 复位
	i3 := e.seedIndicator(t, secs[2], "FI-103") // 翻牌（末端故障）
	e.report(t, i1, domain.IndicatorTriggered, base.Add(time.Minute))
	e.report(t, i2, domain.IndicatorReset, base.Add(2*time.Minute))
	e.report(t, i3, domain.IndicatorTriggered, base.Add(3*time.Minute))

	res, err := e.locate.Locate(LocateInput{FeederID: feederID})
	if err != nil {
		t.Fatal(err)
	}
	// 候选为末端区段 sec3。
	if res.PrimarySectionID != secs[2] {
		t.Fatalf("primary should be %s, got %s", secs[2], res.PrimarySectionID)
	}
	// FI-101 为孤立翻牌，应标记可疑。
	if len(res.Suspicious) != 1 || res.Suspicious[0].IndicatorID != i1 {
		t.Fatalf("expected FI-101 suspicious, got %+v", res.Suspicious)
	}
	// 可疑标记应持久化。
	ind, err := e.store.GetIndicator(i1)
	if err != nil {
		t.Fatal(err)
	}
	if !ind.Suspicious {
		t.Fatal("indicator should be persisted as suspicious")
	}
}

func TestLocate_NoFaultSignal(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	e.seedIndicator(t, secs[0], "FI-101") // 全复位
	if _, err := e.locate.Locate(LocateInput{FeederID: feederID}); !errors.Is(err, domain.ErrNoFaultSignal) {
		t.Fatalf("expected ErrNoFaultSignal, got %v", err)
	}
}

func TestLocate_SuspiciousExcludedFromReasoning(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	base := time.Date(2026, 8, 25, 16, 0, 0, 0, time.Local)
	i1 := e.seedIndicator(t, secs[0], "FI-101")
	i3 := e.seedIndicator(t, secs[2], "FI-103")
	e.report(t, i1, domain.IndicatorTriggered, base.Add(time.Minute))
	if _, err := e.signals.FlagSuspicious(i1, true, "人工核验", "tester", "req"); err != nil {
		t.Fatal(err)
	}
	e.report(t, i3, domain.IndicatorTriggered, base.Add(2*time.Minute))

	res, err := e.locate.Locate(LocateInput{FeederID: feederID})
	if err != nil {
		t.Fatal(err)
	}
	if res.PrimarySectionID != secs[2] {
		t.Fatalf("primary should be %s, got %s", secs[2], res.PrimarySectionID)
	}
	// 已标记可疑的翻牌出现在可疑列表且不参与推理。
	found := false
	for _, sp := range res.Suspicious {
		if sp.IndicatorID == i1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("pre-flagged suspicious should be reported, got %+v", res.Suspicious)
	}
}

func TestLocate_FeederNotFound(t *testing.T) {
	e := newTestEnv(t)
	if _, err := e.locate.Locate(LocateInput{FeederID: "F-NOPE"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
