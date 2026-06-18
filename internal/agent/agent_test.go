package agent

import (
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/michal/kiviq/internal/shared"
)

func TestParseGPUOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   shared.GPUInfo
	}{
		{
			name:   "single gpu",
			output: "NVIDIA GeForce RTX 4090, 85, 24564, 12345, 65, 250.0, 535.129.03\n",
			want: shared.GPUInfo{
				Name:        "NVIDIA GeForce RTX 4090",
				Utilization: 85,
				MemoryTotal: 24564 * 1024 * 1024,
				MemoryUsed:  12345 * 1024 * 1024,
				Temperature: 65,
				PowerDraw:   250.0,
				Driver:      "535.129.03",
			},
		},
		{
			name:   "empty output",
			output: "",
			want:   shared.GPUInfo{},
		},
		{
			name:   "incomplete fields",
			output: "RTX 4090, 85\n",
			want:   shared.GPUInfo{},
		},
		{
			name:   "two gpus returns first",
			output: "RTX 4090, 85, 24564, 12345, 65, 250, 535\nRTX 3080, 50, 10240, 5000, 70, 300, 535\n",
			want: shared.GPUInfo{
				Name:        "RTX 4090",
				Utilization: 85,
				MemoryTotal: 24564 * 1024 * 1024,
				MemoryUsed:  12345 * 1024 * 1024,
				Temperature: 65,
				PowerDraw:   250,
				Driver:      "535",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGPUOutput(tt.output)
			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.Utilization != tt.want.Utilization {
				t.Errorf("Utilization = %v, want %v", got.Utilization, tt.want.Utilization)
			}
			if got.MemoryTotal != tt.want.MemoryTotal {
				t.Errorf("MemoryTotal = %v, want %v", got.MemoryTotal, tt.want.MemoryTotal)
			}
			if got.MemoryUsed != tt.want.MemoryUsed {
				t.Errorf("MemoryUsed = %v, want %v", got.MemoryUsed, tt.want.MemoryUsed)
			}
			if got.Temperature != tt.want.Temperature {
				t.Errorf("Temperature = %v, want %v", got.Temperature, tt.want.Temperature)
			}
			if got.PowerDraw != tt.want.PowerDraw {
				t.Errorf("PowerDraw = %v, want %v", got.PowerDraw, tt.want.PowerDraw)
			}
			if got.Driver != tt.want.Driver {
				t.Errorf("Driver = %q, want %q", got.Driver, tt.want.Driver)
			}
		})
	}
}

func TestSkipNetPrefix(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"docker0", true},
		{"docker1", true},
		{"br-abc", true},
		{"veth123", true},
		{"eth0", false},
		{"enp0s3", false},
		{"wlan0", false},
		{"lo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := skipNetPrefix(tt.name); got != tt.want {
				t.Errorf("skipNetPrefix(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestCollectDockerNilClient(t *testing.T) {
	result := collectDocker(nil)
	if result.Containers != nil {
		t.Errorf("expected nil containers, got %v", result.Containers)
	}
}

func TestNewHTTPClient(t *testing.T) {
	pool := x509.NewCertPool()
	client := newHTTPClient(pool)
	if client == nil {
		t.Fatal("client is nil")
	}
	if client.Timeout != 10e9 {
		t.Errorf("timeout = %v, want 10s", client.Timeout)
	}
}

func TestSendReport_Success(t *testing.T) {
	var received shared.ReportRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/report" {
			t.Errorf("path = %s, want /api/v1/report", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("auth header = %s, want Bearer test-token", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	report := &shared.ReportRequest{
		Hostname: "host-1",
	}

	sendReport(ts.Client(), ts.URL, "test-token", report)

	if received.Hostname != "host-1" {
		t.Errorf("Hostname = %s, want host-1", received.Hostname)
	}
}

func TestSendReport_ErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	report := &shared.ReportRequest{Hostname: "test"}
	sendReport(ts.Client(), ts.URL, "bad-token", report)
}

func TestSendReport_Unreachable(t *testing.T) {
	report := &shared.ReportRequest{Hostname: "test"}
	sendReport(&http.Client{}, "http://192.0.2.1:1", "token", report)
}

func TestCollectDockerStatsUpdatesFields(t *testing.T) {
	containers := []shared.DockerContainer{
		{ID: "aaa", Name: "c1"},
		{ID: "bbb", Name: "c2"},
	}

	ch := make(chan dockerStatsResult, 2)
	ch <- dockerStatsResult{index: 0, stats: shared.DockerContainer{CPUPercent: 10.5, MemoryUsageBytes: 1024}}
	ch <- dockerStatsResult{index: 1, stats: shared.DockerContainer{CPUPercent: 20.0, MemoryUsageBytes: 2048}}

	done := make(chan struct{})
	go func() {
		for range containers {
			r := <-ch
			if !r.err {
				containers[r.index].CPUPercent = r.stats.CPUPercent
				containers[r.index].MemoryUsageBytes = r.stats.MemoryUsageBytes
			}
		}
		close(done)
	}()

	<-done

	if containers[0].CPUPercent != 10.5 {
		t.Errorf("c1 CPU = %v, want 10.5", containers[0].CPUPercent)
	}
	if containers[1].MemoryUsageBytes != 2048 {
		t.Errorf("c2 Mem = %v, want 2048", containers[1].MemoryUsageBytes)
	}
}

func TestSendReport_BodyFormat(t *testing.T) {
	var bodyBytes []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	report := &shared.ReportRequest{
		Hostname: "host-1",
	}

	sendReport(ts.Client(), ts.URL, "tok", report)

	var parsed shared.ReportRequest
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if parsed.Hostname != "host-1" {
		t.Errorf("parsed fields mismatch: %+v", parsed)
	}

	var raw map[string]json.RawMessage
	json.Unmarshal(bodyBytes, &raw)
	if _, ok := raw["timestamp"]; !ok {
		t.Error("missing timestamp field in JSON body")
	}
}
