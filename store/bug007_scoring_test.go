package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"example.com/grid-fault-locate-service/domain"
)

func TestBug007SaveKeepsDataFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	p, err := NewPersister(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Save(snapshot{Feeders: map[string]*domain.Feeder{}}); err != nil {
		t.Fatalf("Save 出错: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Save 成功后数据文件不应被删除: %v", err)
	}
}

func TestBug007SavePropagatesError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "targetdir")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := NewPersister(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Save(snapshot{}); err == nil {
		t.Fatal("Save 写入失败时应返回错误，不应被 defer 吞掉")
	}
}

func TestBug007AuditTruncateKeepsNewest(t *testing.T) {
	st, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2010; i++ {
		if err := st.AppendAudit(&domain.AuditEntry{ID: fmt.Sprintf("AU-%04d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	list := st.ListAudits(0)
	if len(list) != 2000 {
		t.Fatalf("应保留 2000 条，实际 %d", len(list))
	}
	if list[0].ID != "AU-2009" {
		t.Fatalf("应保留最新记录，首条实际 %s", list[0].ID)
	}
}

func TestBug007ListAuditsNewestFirst(t *testing.T) {
	st, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	_ = st.AppendAudit(&domain.AuditEntry{ID: "AU-0"})
	_ = st.AppendAudit(&domain.AuditEntry{ID: "AU-1"})
	list := st.ListAudits(0)
	if list[0].ID != "AU-1" {
		t.Fatalf("审计列表应新→旧，首条实际 %s", list[0].ID)
	}
}
