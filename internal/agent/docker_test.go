package agent

import (
	"net/netip"
	"testing"

	"github.com/moby/moby/api/types/container"
)

func TestSummaryToContainer(t *testing.T) {
	c := container.Summary{
		ID:     "abcdef0123456789",
		Names:  []string{"/web", "/web-alias"},
		Image:  "nginx:latest",
		Status: "Up 3 hours",
		State:  "running",
		Ports: []container.PortSummary{
			{
				IP:          netip.MustParseAddr("127.0.0.1"),
				PrivatePort: 80,
				PublicPort:  8080,
				Type:        "tcp",
			},
		},
	}

	dc := summaryToContainer(c)

	if dc.ID != "abcdef012345" {
		t.Errorf("ID = %q, want truncated to 12 chars", dc.ID)
	}
	if dc.Name != "/web" {
		t.Errorf("Name = %q, want first name", dc.Name)
	}
	if dc.Image != "nginx:latest" {
		t.Errorf("Image = %q", dc.Image)
	}
	if dc.Status != "Up 3 hours" {
		t.Errorf("Status = %q", dc.Status)
	}
	// State must come straight from the list summary, not an inspect call.
	if dc.State != "running" {
		t.Errorf("State = %q, want %q from Summary.State", dc.State, "running")
	}

	// Verify Ports mapping
	if len(dc.Ports) != 1 {
		t.Fatalf("len(Ports) = %d, want 1", len(dc.Ports))
	}
	p := dc.Ports[0]
	if p.IP != "127.0.0.1" {
		t.Errorf("Port IP = %q, want 127.0.0.1", p.IP)
	}
	if p.PrivatePort != 80 {
		t.Errorf("Port PrivatePort = %d, want 80", p.PrivatePort)
	}
	if p.PublicPort != 8080 {
		t.Errorf("Port PublicPort = %d, want 8080", p.PublicPort)
	}
	if p.Type != "tcp" {
		t.Errorf("Port Type = %q, want tcp", p.Type)
	}
}

func TestSummaryToContainerNoNamesShortID(t *testing.T) {
	dc := summaryToContainer(container.Summary{ID: "abc", State: "exited"})

	if dc.ID != "abc" {
		t.Errorf("ID = %q, want short ID passed through without panic", dc.ID)
	}
	if dc.Name != "" {
		t.Errorf("Name = %q, want empty when no names", dc.Name)
	}
	if dc.State != "exited" {
		t.Errorf("State = %q", dc.State)
	}
}
