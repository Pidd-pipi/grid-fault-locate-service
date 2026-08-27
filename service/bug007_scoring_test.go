package service

import (
	"testing"

	"example.com/grid-fault-locate-service/domain"
)

func TestBug007RecordSucceeds(t *testing.T) {
	e := newTestEnv(t)
	if err := e.audit.Record("req-1", domain.AuditFeederCreate, "feeder", "F1", "op", "create"); err != nil {
		t.Fatalf("Record 应成功，实际 %v", err)
	}
}
