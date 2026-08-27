package service

import (
	"errors"
	"testing"
	"time"

	"example.com/grid-fault-locate-service/domain"
)

func TestBug003NoSignalChainPreserved(t *testing.T) {
	e := newTestEnv(t)
	feederID, _ := e.seedChain(t)
	_, err := e.locate.Locate(LocateInput{FeederID: feederID})
	if err == nil {
		t.Fatal("无翻牌信号应报错")
	}
	if !errors.Is(err, domain.ErrNoFaultSignal) {
		t.Fatalf("错误链应保留 ErrNoFaultSignal，实际 %v", err)
	}
}

func TestBug003SuspiciousChainPreserved(t *testing.T) {
	e := newTestEnv(t)
	feederID, secs := e.seedChain(t)
	indID := e.seedIndicator(t, secs[0], "FI-X")
	at := time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local)
	e.report(t, indID, domain.IndicatorTriggered, at)
	if _, err := e.signals.FlagSuspicious(indID, true, "manual", "op", "req"); err != nil {
		t.Fatal(err)
	}
	_, err := e.locate.Locate(LocateInput{FeederID: feederID})
	if err == nil {
		t.Fatal("仅可疑信号应报错")
	}
	if !errors.Is(err, domain.ErrSuspiciousOnly) {
		t.Fatalf("错误链应保留 ErrSuspiciousOnly，实际 %v", err)
	}
}
