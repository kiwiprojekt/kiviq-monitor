package shared

import "time"

type TempInfo struct {
	Label       string  `json:"label"`
	Celsius     float64 `json:"celsius"`
	HighCelsius float64 `json:"high_celsius,omitempty"`
}

type CPUInfo struct {
	Cores        int        `json:"cores"`
	UsagePercent float64    `json:"usage_percent"`
	PerCore      []float64  `json:"per_core"`
	ModelName    string     `json:"model_name,omitempty"`
	FreqMHz      []float64  `json:"freq_mhz,omitempty"`
	FreqMinMHz   float64    `json:"freq_min_mhz,omitempty"`
	FreqMaxMHz   float64    `json:"freq_max_mhz,omitempty"`
	Temperatures []TempInfo `json:"temperatures,omitempty"`
}

type MemoryInfo struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
	SwapTotal    uint64  `json:"swap_total,omitempty"`
	SwapUsed     uint64  `json:"swap_used,omitempty"`
	SwapFree     uint64  `json:"swap_free,omitempty"`
}

type DiskInfo struct {
	Device       string  `json:"device"`
	Mount        string  `json:"mount"`
	Fstype       string  `json:"fstype"`
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
	ReadBytes    uint64  `json:"read_bytes"`
	WriteBytes   uint64  `json:"write_bytes"`
	ReadCount    uint64  `json:"read_count"`
	WriteCount   uint64  `json:"write_count"`
}

type NetworkInfo struct {
	Interface  string `json:"interface"`
	MAC        string `json:"mac"`
	IP         string `json:"ip"`
	MTU        int    `json:"mtu"`
	SpeedMbps  int    `json:"speed_mbps"`
	BytesIn    uint64 `json:"bytes_in"`
	BytesOut   uint64 `json:"bytes_out"`
	PacketsIn  uint64 `json:"packets_in"`
	PacketsOut uint64 `json:"packets_out"`
	ErrorsIn   uint64 `json:"errors_in"`
	ErrorsOut  uint64 `json:"errors_out"`
}

type DockerPort struct {
	IP          string `json:"ip,omitempty"`
	PrivatePort uint16 `json:"private_port"`
	PublicPort  uint16 `json:"public_port,omitempty"`
	Type        string `json:"type"`
}

type DockerContainer struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Image            string       `json:"image"`
	Status           string       `json:"status"`
	State            string       `json:"state"`
	CPUPercent       float64      `json:"cpu_percent"`
	MemoryUsageBytes uint64       `json:"memory_usage_bytes"`
	MemoryLimitBytes uint64       `json:"memory_limit_bytes"`
	MemoryPercent    float64      `json:"memory_percent"`
	NetworkRxBytes   uint64       `json:"network_rx_bytes"`
	NetworkTxBytes   uint64       `json:"network_tx_bytes"`
	Ports            []DockerPort `json:"ports,omitempty"`
}

type DockerInfo struct {
	Containers []DockerContainer `json:"containers"`
}

type GPUInfo struct {
	Name        string  `json:"name"`
	Utilization float64 `json:"utilization_percent"`
	MemoryTotal uint64  `json:"memory_total_bytes"`
	MemoryUsed  uint64  `json:"memory_used_bytes"`
	Temperature float64 `json:"temperature_celsius"`
	PowerDraw   float64 `json:"power_draw_watts"`
	Driver      string  `json:"driver_version"`
}

type SystemInfo struct {
	OS           string  `json:"os"`
	Platform     string  `json:"platform"`
	PlatformVer  string  `json:"platform_version"`
	Kernel       string  `json:"kernel"`
	Arch         string  `json:"arch"`
	VirtSystem   string  `json:"virt_system"`
	VirtRole     string  `json:"virt_role"`
	Load1        float64 `json:"load1,omitempty"`
	Load5        float64 `json:"load5,omitempty"`
	Load15       float64 `json:"load15,omitempty"`
	ProcessCount int     `json:"process_count,omitempty"`
}

type ReportRequest struct {
	Hostname      string        `json:"hostname"`
	System        SystemInfo    `json:"system"`
	Timestamp     time.Time     `json:"timestamp"`
	UptimeSeconds uint64        `json:"uptime_seconds"`
	CPU           CPUInfo       `json:"cpu"`
	Memory        MemoryInfo    `json:"memory"`
	Disk          []DiskInfo    `json:"disk"`
	Network       []NetworkInfo `json:"network"`
	Docker        DockerInfo    `json:"docker"`
	GPU           GPUInfo       `json:"gpu,omitempty"`
}

type ReportResponse struct {
	Status string `json:"status"`
}

type AgentSnapshot struct {
	// AgentID and AgentName are assigned by the monitor from the authenticated
	// token, never by the agent — an agent cannot declare its own identity.
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
	ReportRequest
	// LastSeen is the authoritative liveness signal. Clients derive online-ness
	// from it against their own clock — a pushed "online" flag would freeze at
	// the last received tick and never flip false once an agent goes silent, so
	// the monitor deliberately ships no such flag.
	LastSeen time.Time `json:"last_seen"`
	Order    int       `json:"order,omitempty"`
}

func SnapshotFromRequest(id, name string, req ReportRequest) AgentSnapshot {
	return AgentSnapshot{
		AgentID:       id,
		AgentName:     name,
		ReportRequest: req,
		LastSeen:      time.Now(),
	}
}

func (s *AgentSnapshot) ComputeHistoryPoint() HistoryPoint {
	var rx, tx, dr, dw uint64
	for _, n := range s.Network {
		rx += n.BytesIn
		tx += n.BytesOut
	}
	for _, d := range s.Disk {
		dr += d.ReadBytes
		dw += d.WriteBytes
	}
	return HistoryPoint{
		Timestamp: s.LastSeen.Unix(),
		CPU:       s.CPU.UsagePercent,
		Memory:    s.Memory.UsagePercent,
		NetworkRx: rx,
		NetworkTx: tx,
		DiskRead:  dr,
		DiskWrite: dw,
	}
}

type HistoryPoint struct {
	Timestamp int64   `json:"t"`
	CPU       float64 `json:"cpu"`
	Memory    float64 `json:"mem"`
	NetworkRx uint64  `json:"rx"`
	NetworkTx uint64  `json:"tx"`
	DiskRead  uint64  `json:"dr"`
	DiskWrite uint64  `json:"dw"`
}

type HistoryResponse struct {
	AgentID string         `json:"agent_id"`
	Points  []HistoryPoint `json:"points"`
}

type WSMessage struct {
	Type    string         `json:"type"`
	Data    *AgentSnapshot `json:"data"`
	History *HistoryPoint  `json:"history,omitempty"`
}
