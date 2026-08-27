package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/grid-fault-locate-service/domain"
)

func TestCorruptFileBackedUpAndEmptyStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.ListFeeders()) != 0 {
		t.Fatalf("corrupt file should degrade to empty store, got %d feeders", len(st.ListFeeders()))
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("corrupt file should be backed up to .bak: %v", err)
	}
}

func TestAtomicSaveLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	st, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	f := domain.NewFeeder(st.NextID("F"), "测试线", "变电站", "10kV", domain.FeederActive, time.Now())
	if err := st.CreateFeeder(f); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "data.json" {
			t.Fatalf("unexpected file left after atomic save: %s", e.Name())
		}
	}
}
