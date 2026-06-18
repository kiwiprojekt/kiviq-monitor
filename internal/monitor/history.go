package monitor

import (
	"sort"
	"time"

	"github.com/michal/kiviq/internal/shared"
)

// DefaultHistoryPoints is the per-agent history cap when none is configured:
// 3600 points, i.e. one hour at the default 1s report interval.
const DefaultHistoryPoints = 3600

// HistoryBuffer is not safe for concurrent use on its own. Its only owner is
// Store, which already serializes every Append/Get under its own RWMutex, so a
// second lock here would be pure double-locking.
type HistoryBuffer struct {
	points map[string][]shared.HistoryPoint
	max    int
}

// NewHistoryBuffer caps each agent's history at max points; max <= 0 uses
// DefaultHistoryPoints.
func NewHistoryBuffer(max int) *HistoryBuffer {
	if max <= 0 {
		max = DefaultHistoryPoints
	}
	return &HistoryBuffer{
		points: make(map[string][]shared.HistoryPoint),
		max:    max,
	}
}

// Append stores and returns the history point for snap, so the caller can reuse
// it (e.g. for the live broadcast) without recomputing.
func (h *HistoryBuffer) Append(agentID string, snap shared.AgentSnapshot) shared.HistoryPoint {
	pt := snap.ComputeHistoryPoint()

	h.points[agentID] = append(h.points[agentID], pt)
	if len(h.points[agentID]) > h.max {
		h.points[agentID] = h.points[agentID][len(h.points[agentID])-h.max:]
	}
	return pt
}

func (h *HistoryBuffer) Get(agentID string, since time.Time) []shared.HistoryPoint {
	pts := h.points[agentID]
	if len(pts) == 0 {
		return nil
	}

	sinceUnix := since.Unix()

	start := sort.Search(len(pts), func(i int) bool {
		return pts[i].Timestamp >= sinceUnix
	})

	result := make([]shared.HistoryPoint, len(pts)-start)
	copy(result, pts[start:])
	return result
}
