package monitor

import (
	"testing"
	"time"

	"github.com/michal/kiviq/internal/shared"
)

func TestHistoryBufferAppendAndGet(t *testing.T) {
	h := NewHistoryBuffer(0)

	snap := shared.AgentSnapshot{
		ReportRequest: shared.ReportRequest{
			CPU:    shared.CPUInfo{UsagePercent: 42.5},
			Memory: shared.MemoryInfo{UsagePercent: 60.0},
			Network: []shared.NetworkInfo{
				{BytesIn: 1000, BytesOut: 500},
			},
			Disk: []shared.DiskInfo{
				{ReadBytes: 2000, WriteBytes: 1000},
			},
		},
		LastSeen: time.Now(),
	}

	h.Append("a1", snap)

	points := h.Get("a1", time.Now().Add(-1*time.Minute))
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}

	p := points[0]
	if p.CPU != 42.5 {
		t.Errorf("CPU = %v, want 42.5", p.CPU)
	}
	if p.Memory != 60.0 {
		t.Errorf("Memory = %v, want 60.0", p.Memory)
	}
	if p.NetworkRx != 1000 {
		t.Errorf("NetworkRx = %d, want 1000", p.NetworkRx)
	}
	if p.NetworkTx != 500 {
		t.Errorf("NetworkTx = %d, want 500", p.NetworkTx)
	}
	if p.DiskRead != 2000 {
		t.Errorf("DiskRead = %d, want 2000", p.DiskRead)
	}
	if p.DiskWrite != 1000 {
		t.Errorf("DiskWrite = %d, want 1000", p.DiskWrite)
	}
}

func TestHistoryBufferMaxSize(t *testing.T) {
	h := NewHistoryBuffer(0)

	now := time.Now()
	for i := 0; i < 3601; i++ {
		snap := shared.AgentSnapshot{
			ReportRequest: shared.ReportRequest{
				CPU: shared.CPUInfo{UsagePercent: float64(i)},
			},
			LastSeen: now.Add(time.Duration(i) * time.Second),
		}
		h.Append("a1", snap)
	}

	points := h.Get("a1", time.Time{}) // since epoch = all points
	if len(points) != 3600 {
		t.Errorf("expected 3600 points (max), got %d", len(points))
	}

	// Oldest point should be at index 1 (0 was evicted)
	if points[0].CPU != 1.0 {
		t.Errorf("oldest CPU = %v, want 1.0 (index 0 evicted)", points[0].CPU)
	}
}

func TestHistoryBufferConfigurableMax(t *testing.T) {
	h := NewHistoryBuffer(5)

	now := time.Now()
	for i := 0; i < 10; i++ {
		h.Append("a1", shared.AgentSnapshot{
			ReportRequest: shared.ReportRequest{CPU: shared.CPUInfo{UsagePercent: float64(i)}},
			LastSeen:      now.Add(time.Duration(i) * time.Second),
		})
	}

	points := h.Get("a1", time.Time{})
	if len(points) != 5 {
		t.Fatalf("expected 5 points (configured max), got %d", len(points))
	}
	if points[0].CPU != 5.0 {
		t.Errorf("oldest CPU = %v, want 5.0 (points 0-4 evicted)", points[0].CPU)
	}
}

func TestHistoryBufferGetSince(t *testing.T) {
	h := NewHistoryBuffer(0)

	now := time.Now()
	for i := 0; i < 10; i++ {
		snap := shared.AgentSnapshot{
			ReportRequest: shared.ReportRequest{
				CPU: shared.CPUInfo{UsagePercent: float64(i)},
			},
			LastSeen: now.Add(-time.Duration(10-i) * time.Minute),
		}
		h.Append("a1", snap)
	}

	// Get last 5 minutes
	points := h.Get("a1", now.Add(-5*time.Minute))
	if len(points) != 5 { // -5, -4, -3, -2, -1 minutes ago = 5 points
		t.Errorf("expected 5 points, got %d", len(points))
	}
}

func TestHistoryBufferEmpty(t *testing.T) {
	h := NewHistoryBuffer(0)

	points := h.Get("nonexistent", time.Time{})
	if len(points) != 0 {
		t.Errorf("expected 0 points, got %d", len(points))
	}
}

func TestHistoryBufferMultipleAgents(t *testing.T) {
	h := NewHistoryBuffer(0)

	now := time.Now()
	h.Append("a1", shared.AgentSnapshot{ReportRequest: shared.ReportRequest{CPU: shared.CPUInfo{UsagePercent: 10}}, LastSeen: now})
	h.Append("a2", shared.AgentSnapshot{ReportRequest: shared.ReportRequest{CPU: shared.CPUInfo{UsagePercent: 20}}, LastSeen: now})

	p1 := h.Get("a1", time.Time{})
	p2 := h.Get("a2", time.Time{})

	if len(p1) != 1 || len(p2) != 1 {
		t.Errorf("expected 1 point each, got %d and %d", len(p1), len(p2))
	}
	if p1[0].CPU != 10 {
		t.Errorf("a1 CPU = %v, want 10", p1[0].CPU)
	}
	if p2[0].CPU != 20 {
		t.Errorf("a2 CPU = %v, want 20", p2[0].CPU)
	}
}
