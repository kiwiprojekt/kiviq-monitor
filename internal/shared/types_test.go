package shared

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCPUInfoJSON(t *testing.T) {
	info := CPUInfo{
		Cores:        8,
		UsagePercent: 42.5,
		PerCore:      []float64{10, 20, 30, 40, 50, 60, 70, 80},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got CPUInfo
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Cores != 8 {
		t.Errorf("Cores = %d, want 8", got.Cores)
	}
	if got.UsagePercent != 42.5 {
		t.Errorf("UsagePercent = %v, want 42.5", got.UsagePercent)
	}
	if len(got.PerCore) != 8 {
		t.Errorf("PerCore len = %d, want 8", len(got.PerCore))
	}
}

func TestReportRequestJSON(t *testing.T) {
	req := ReportRequest{
		Hostname: "host-1",
		System:   SystemInfo{OS: "ubuntu", Arch: "amd64"},
		CPU:      CPUInfo{Cores: 4},
		Memory:   MemoryInfo{TotalBytes: 8192},
		Disk:     []DiskInfo{{Device: "/dev/sda1", Mount: "/"}},
		Network:  []NetworkInfo{{Interface: "eth0", MAC: "aa:bb:cc:dd:ee:ff"}},
		Docker:   DockerInfo{Containers: []DockerContainer{{ID: "abc123", Name: "web"}}},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ReportRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Hostname != "host-1" {
		t.Errorf("Hostname = %q, want %q", got.Hostname, "host-1")
	}
	if got.System.OS != "ubuntu" {
		t.Errorf("OS = %q, want %q", got.System.OS, "ubuntu")
	}
	if len(got.Disk) != 1 {
		t.Errorf("Disk len = %d, want 1", len(got.Disk))
	}
	if len(got.Network) != 1 {
		t.Errorf("Network len = %d, want 1", len(got.Network))
	}
	if len(got.Docker.Containers) != 1 {
		t.Errorf("Docker.Containers len = %d, want 1", len(got.Docker.Containers))
	}
}

func TestHistoryPointJSON(t *testing.T) {
	pt := HistoryPoint{
		Timestamp: 1234567890,
		CPU:       42.5,
		Memory:    60.0,
		NetworkRx: 1000,
		NetworkTx: 500,
		DiskRead:  2000,
		DiskWrite: 1000,
	}

	data, err := json.Marshal(pt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got HistoryPoint
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Timestamp != 1234567890 {
		t.Errorf("Timestamp = %d, want 1234567890", got.Timestamp)
	}
	if got.CPU != 42.5 {
		t.Errorf("CPU = %v, want 42.5", got.CPU)
	}
}

func TestWSMessageJSON(t *testing.T) {
	msg := WSMessage{
		Type: "snapshot",
		Data: &AgentSnapshot{
			AgentID: "a1",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got WSMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Type != "snapshot" {
		t.Errorf("Type = %q, want %q", got.Type, "snapshot")
	}
	if got.Data == nil || got.Data.AgentID != "a1" {
		t.Errorf("Data.AgentID = %v, want a1", got.Data)
	}
}

func TestAgentSnapshotJSON(t *testing.T) {
	snap := AgentSnapshot{
		AgentID:   "a1",
		AgentName: "web",
		ReportRequest: ReportRequest{
			UptimeSeconds: 3600,
			CPU:           CPUInfo{Cores: 4, UsagePercent: 50},
			Memory:        MemoryInfo{TotalBytes: 8192, UsagePercent: 75},
		},
		LastSeen: time.Unix(1_700_000_000, 0),
	}

	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got AgentSnapshot
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !got.LastSeen.Equal(snap.LastSeen) {
		t.Errorf("LastSeen = %v, want %v", got.LastSeen, snap.LastSeen)
	}
	if got.UptimeSeconds != 3600 {
		t.Errorf("UptimeSeconds = %d, want 3600", got.UptimeSeconds)
	}
}

func TestGPUInfoJSON(t *testing.T) {
	gpu := GPUInfo{
		Name:        "NVIDIA GeForce RTX 3080",
		Utilization: 85.0,
		MemoryTotal: 10737418240,
		MemoryUsed:  8589934592,
		Temperature: 72.0,
		PowerDraw:   280.0,
		Driver:      "535.129.03",
	}

	data, err := json.Marshal(gpu)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got GPUInfo
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != "NVIDIA GeForce RTX 3080" {
		t.Errorf("Name = %q, want %q", got.Name, "NVIDIA GeForce RTX 3080")
	}
	if got.Utilization != 85.0 {
		t.Errorf("Utilization = %v, want 85.0", got.Utilization)
	}
}

func TestTempInfoJSON(t *testing.T) {
	temp := TempInfo{
		Label:       "coretemp",
		Celsius:     65.0,
		HighCelsius: 100.0,
	}

	data, err := json.Marshal(temp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got TempInfo
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Label != "coretemp" {
		t.Errorf("Label = %q, want %q", got.Label, "coretemp")
	}
	if got.HighCelsius != 100.0 {
		t.Errorf("HighCelsius = %v, want 100.0", got.HighCelsius)
	}
}

func TestCPUInfoWithNewFields(t *testing.T) {
	info := CPUInfo{
		Cores:        8,
		UsagePercent: 42.5,
		PerCore:      []float64{10, 20, 30, 40, 50, 60, 70, 80},
		ModelName:    "Intel Core i7-12700K",
		FreqMHz:      []float64{3600, 3600, 3600, 3600, 3600, 3600, 3600, 3600},
		FreqMinMHz:   800,
		FreqMaxMHz:   5000,
		Temperatures: []TempInfo{{Label: "coretemp", Celsius: 65}},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got CPUInfo
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ModelName != "Intel Core i7-12700K" {
		t.Errorf("ModelName = %q, want %q", got.ModelName, "Intel Core i7-12700K")
	}
	if got.FreqMaxMHz != 5000 {
		t.Errorf("FreqMaxMHz = %v, want 5000", got.FreqMaxMHz)
	}
	if len(got.Temperatures) != 1 {
		t.Errorf("Temperatures len = %d, want 1", len(got.Temperatures))
	}
}

func TestMemoryInfoWithSwap(t *testing.T) {
	mem := MemoryInfo{
		TotalBytes:   8589934592,
		UsedBytes:    4294967296,
		FreeBytes:    4294967296,
		UsagePercent: 50.0,
		SwapTotal:    2147483648,
		SwapUsed:     1073741824,
		SwapFree:     1073741824,
	}

	data, err := json.Marshal(mem)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got MemoryInfo
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.SwapTotal != 2147483648 {
		t.Errorf("SwapTotal = %d, want 2147483648", got.SwapTotal)
	}
	if got.SwapUsed != 1073741824 {
		t.Errorf("SwapUsed = %d, want 1073741824", got.SwapUsed)
	}
}
