package monitor

import (
	"sync"
	"testing"
	"time"

	"github.com/michal/kiviq/internal/shared"
)

func TestStoreUpdateAndGet(t *testing.T) {
	store := NewStore(0)

	req := shared.ReportRequest{
		Hostname: "host-1",
		System:   shared.SystemInfo{OS: "ubuntu", Arch: "amd64"},
		CPU:      shared.CPUInfo{Cores: 4, UsagePercent: 42.5},
		Memory:   shared.MemoryInfo{TotalBytes: 8 * 1024 * 1024 * 1024, UsagePercent: 65.0},
	}

	store.Update("agent-1", "web-agent", req)

	snap, ok := store.Get("agent-1")
	if !ok {
		t.Fatal("expected to find agent-1")
	}
	if snap.AgentName != "web-agent" {
		t.Errorf("AgentName = %q, want %q", snap.AgentName, "web-agent")
	}
	if snap.System.OS != "ubuntu" {
		t.Errorf("OS = %q, want %q", snap.System.OS, "ubuntu")
	}
	if snap.CPU.UsagePercent != 42.5 {
		t.Errorf("CPU = %v, want 42.5", snap.CPU.UsagePercent)
	}
}

func TestStoreGetNotFound(t *testing.T) {
	store := NewStore(0)

	_, ok := store.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestStoreGetAll(t *testing.T) {
	store := NewStore(0)

	store.Update("a1", "agent-1", shared.ReportRequest{})
	store.Update("a2", "agent-2", shared.ReportRequest{})

	agents := store.GetAll()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}

	names := map[string]bool{}
	for _, s := range agents {
		names[s.AgentName] = true
	}
	if !names["agent-1"] || !names["agent-2"] {
		t.Errorf("missing agents: %v", names)
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	store := NewStore(0)
	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				store.Update("agent", "agent", shared.ReportRequest{
					CPU: shared.CPUInfo{UsagePercent: float64(j)},
				})
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				store.GetAll()
				store.Get("agent")
				store.GetHistory("agent", time.Now().Add(-1*time.Hour))
			}
		}()
	}

	wg.Wait()
}

func TestStoreGetHistory(t *testing.T) {
	store := NewStore(0)

	store.Update("a1", "a1", shared.ReportRequest{
		CPU: shared.CPUInfo{UsagePercent: 10},
	})
	store.Update("a1", "a1", shared.ReportRequest{
		CPU: shared.CPUInfo{UsagePercent: 20},
	})

	points := store.GetHistory("a1", time.Now().Add(-1*time.Hour))
	if len(points) != 2 {
		t.Fatalf("expected 2 history points, got %d", len(points))
	}
	if points[0].CPU != 10 {
		t.Errorf("first point CPU = %v, want 10", points[0].CPU)
	}
	if points[1].CPU != 20 {
		t.Errorf("second point CPU = %v, want 20", points[1].CPU)
	}
}

func TestStoreGetHistoryEmpty(t *testing.T) {
	store := NewStore(0)

	points := store.GetHistory("nonexistent", time.Now().Add(-1*time.Hour))
	if points != nil {
		t.Errorf("expected nil for nonexistent agent, got %v", points)
	}
}
