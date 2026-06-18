package monitor

import (
	"sync"
	"time"

	"github.com/michal/kiviq/internal/shared"
)

type Store struct {
	mu        sync.RWMutex
	snapshots map[string]*shared.AgentSnapshot
	history   *HistoryBuffer
}

// NewStore creates a Store whose per-agent history is capped at historyPoints;
// historyPoints <= 0 uses DefaultHistoryPoints.
func NewStore(historyPoints int) *Store {
	return &Store{
		snapshots: make(map[string]*shared.AgentSnapshot),
		history:   NewHistoryBuffer(historyPoints),
	}
}

// Update records a report under the given agent id and display name, both of
// which the caller derives from the authenticated token — never from the
// report body. Liveness is the client's concern (see AgentSnapshot.LastSeen),
// so the store keeps no online flag.
func (s *Store) Update(id, name string, req shared.ReportRequest) (*shared.AgentSnapshot, shared.HistoryPoint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap := shared.SnapshotFromRequest(id, name, req)
	s.snapshots[id] = &snap
	pt := s.history.Append(id, snap)
	return &snap, pt
}

func (s *Store) GetHistory(agentID string, since time.Time) []shared.HistoryPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.history.Get(agentID, since)
}

func (s *Store) GetAll() []shared.AgentSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]shared.AgentSnapshot, 0, len(s.snapshots))
	for _, snap := range s.snapshots {
		result = append(result, *snap)
	}
	return result
}

func (s *Store) Get(id string) (*shared.AgentSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap, ok := s.snapshots[id]
	if !ok {
		return nil, false
	}
	// Copy so callers can't mutate the stored snapshot through the pointer.
	cp := *snap
	return &cp, true
}
