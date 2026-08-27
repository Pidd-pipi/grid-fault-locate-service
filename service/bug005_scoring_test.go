package service

import (
	"errors"
	"testing"
	"time"

	"example.com/grid-fault-locate-service/domain"
)

func TestBug005ReportSignalNotFoundChain(t *testing.T) {
	e := newTestEnv(t)
	_, err := e.signals.ReportSignal("FI-missing", domain.IndicatorTriggered, time.Now(), "op", "req")
	if err == nil {
		t.Fatal("上报不存在的指示器应报错")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("错误链应保留 ErrNotFound，实际 %v", err)
	}
}

func TestBug005UpdateIndicatorNotFoundChain(t *testing.T) {
	e := newTestEnv(t)
	_, err := e.signals.UpdateIndicator("FI-missing", "x", 0.5, "op", "req")
	if err == nil {
		t.Fatal("更新不存在的指示器应报错")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("错误链应保留 ErrNotFound，实际 %v", err)
	}
}

func TestBug005DeleteIndicatorNotFoundChain(t *testing.T) {
	e := newTestEnv(t)
	err := e.signals.DeleteIndicator("FI-missing", "op", "req")
	if err == nil {
		t.Fatal("删除不存在的指示器应报错")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("错误链应保留 ErrNotFound，实际 %v", err)
	}
}
