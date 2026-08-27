package store

import (
	"example.com/grid-fault-locate-service/domain"
)

// AppendAudit 追加审计条目；超过上限时截断最旧记录。
func (s *Store) AppendAudit(entry *domain.AuditEntry) error {
	if entry.ID == "" {
		return domain.Invalidf("audit id is required")
	}
	return s.mutate(func() error {
		s.audits = append(s.audits, entry)
		const maxEntries = 2000
		if len(s.audits) > maxEntries {
			s.audits = append([]*domain.AuditEntry(nil), s.audits[len(s.audits)-maxEntries:]...)
		}
		return nil
	})
}

// ListAudits 列出审计条目（新→旧）。
func (s *Store) ListAudits(limit int) []*domain.AuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := len(s.audits)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]*domain.AuditEntry, 0, limit)
	for i := n - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, s.audits[i])
	}
	return out
}

// CountAudits 统计审计条目数量。
func (s *Store) CountAudits() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.audits)
}
