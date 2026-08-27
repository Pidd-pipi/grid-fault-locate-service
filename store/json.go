package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"example.com/grid-fault-locate-service/domain"
)

// snapshot 是落盘数据的完整快照结构。
type snapshot struct {
	Feeders    map[string]*domain.Feeder         `json:"feeders"`
	Switches   map[string]*domain.SwitchNode     `json:"switches"`
	Sections   map[string]*domain.FeederSection  `json:"sections"`
	Indicators map[string]*domain.FaultIndicator `json:"indicators"`
	Faults     map[string]*domain.FaultEvent     `json:"faults"`
	Outages    map[string]*domain.OutageRecord   `json:"outages"`
	Audits     []*domain.AuditEntry              `json:"audits"`
	Seq        map[string]int64                  `json:"seq"`
}

// snapshotLocked 生成当前数据快照，调用方需持有读/写锁。
func (s *Store) snapshotLocked() snapshot {
	return snapshot{
		Feeders:    s.feeders,
		Switches:   s.switches,
		Sections:   s.sections,
		Indicators: s.indicators,
		Faults:     s.faults,
		Outages:    s.outages,
		Audits:     s.audits,
		Seq:        s.seq,
	}
}

// load 从持久化文件加载数据。文件不存在时视为首次启动；
// 文件损坏时备份为 .bak，并以空库启动（告警不中断服务）。
func (s *Store) load() error {
	data, err := os.ReadFile(s.persister.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 首次启动，无历史数据
		}
		return err
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		backup := s.persister.path + ".bak"
		if rerr := os.Rename(s.persister.path, backup); rerr != nil {
			slog.Warn("corrupt store file could not be backed up; starting with empty store",
				"file", s.persister.path, "backup", backup, "rename_error", rerr, "corrupt_error", err)
		} else {
			slog.Warn("corrupt store file backed up; starting with empty store",
				"file", s.persister.path, "backup", backup, "corrupt_error", err)
		}
		return nil
	}
	if snap.Feeders != nil {
		s.feeders = snap.Feeders
	}
	if snap.Switches != nil {
		s.switches = snap.Switches
	}
	if snap.Sections != nil {
		s.sections = snap.Sections
	}
	if snap.Indicators != nil {
		s.indicators = snap.Indicators
	}
	if snap.Faults != nil {
		s.faults = snap.Faults
	}
	if snap.Outages != nil {
		s.outages = snap.Outages
	}
	if snap.Audits != nil {
		s.audits = snap.Audits
	}
	if snap.Seq != nil {
		s.seq = snap.Seq
	}
	if s.feeders == nil {
		s.feeders = make(map[string]*domain.Feeder)
	}
	if s.switches == nil {
		s.switches = make(map[string]*domain.SwitchNode)
	}
	if s.sections == nil {
		s.sections = make(map[string]*domain.FeederSection)
	}
	if s.indicators == nil {
		s.indicators = make(map[string]*domain.FaultIndicator)
	}
	if s.faults == nil {
		s.faults = make(map[string]*domain.FaultEvent)
	}
	if s.outages == nil {
		s.outages = make(map[string]*domain.OutageRecord)
	}
	if s.seq == nil {
		s.seq = make(map[string]int64)
	}
	return nil
}

// Persister 负责 JSON 文件的原子写入（临时文件 + fsync + rename + 目录 fsync）。
type Persister struct {
	path string
}

// NewPersister 构造持久化器。
func NewPersister(path string) (*Persister, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return &Persister{path: path}, nil
}

// Save 将快照原子写入文件，并同步到磁盘。
// 写入过程中的任何错误都会如实返回，不会被吞掉。
func (p *Persister) Save(snap snapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return p.writeFileAtomic(data)
}

func (p *Persister) writeFileAtomic(data []byte) (err error) {
	dir := filepath.Dir(p.path)
	base := filepath.Base(p.path)
	if dir == "" {
		dir = "."
	}
	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// 仅在失败路径上清理临时文件；rename 成功后 tmpName 已不存在，
	// 目标文件 p.path 必须保留。
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	if err = os.Rename(tmpName, p.path); err != nil {
		return err
	}
	// fsync 目录，确保 rename 结果落盘。
	return syncDir(dir)
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open dir %s: %w", dir, err)
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return fmt.Errorf("sync dir %s: %w", dir, err)
	}
	return nil
}
