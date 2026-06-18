package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/michal/kiviq/internal/shared"
	"github.com/moby/moby/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	gopsutilnet "github.com/shirou/gopsutil/v3/net"
)

var skipFstype = map[string]bool{
	"proc": true, "sysfs": true, "devpts": true, "tmpfs": true,
	"cgroup": true, "cgroup2": true, "pstore": true, "securityfs": true,
	"debugfs": true, "hugetlbfs": true, "mqueue": true, "configfs": true,
	"fusectl": true, "autofs": true, "tracefs": true, "bpf": true,
	"overlay": true, "nsfs": true, "rpc_pipefs": true, "nfsd": true,
}

var skipNetPrefixes = []string{"docker", "br-", "veth"}

func skipNetPrefix(name string) bool {
	for _, prefix := range skipNetPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

var parentRe = regexp.MustCompile(`\d+$`)

// hostRoot returns the prefix under which the real host's filesystem is
// readable, or "" when the agent reads its own namespace directly. In sandboxes
// that share the host PID namespace but not its mounts — Home Assistant add-ons
// with host_pid — the host root is reachable at /proc/1/root (set via
// AGENT_HOST_ROOT), so host identity can be read without bind mounts.
func hostRoot() string {
	return os.Getenv("AGENT_HOST_ROOT")
}

// collectHostname reports the host's name. os.Hostname() returns the UTS
// namespace name, which in a container is the container's — so when a host root
// is configured, read the host's /etc/hostname instead.
func collectHostname() string {
	if root := hostRoot(); root != "" {
		if data, err := os.ReadFile(filepath.Join(root, "etc", "hostname")); err == nil {
			if name := strings.TrimSpace(string(data)); name != "" {
				return name
			}
		}
	}
	name, _ := os.Hostname()
	return name
}

func CollectStats(dockerClient *client.Client) (*shared.ReportRequest, error) {
	// The agent does not declare its own ID or name — the monitor derives both
	// from the token the report is authenticated with. Only the hostname is
	// reported, as informational host detail.
	hostname := collectHostname()

	type cpuData struct {
		percent []float64
		model   string
		freqMin float64
		freqMax float64
		freqPer []float64
		temps   []shared.TempInfo
	}
	type sysData struct {
		info   shared.SystemInfo
		uptime uint64
		mem    shared.MemoryInfo
	}

	var cpuR cpuData
	var sysR sysData

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		percent, _ := cpu.Percent(time.Second, true)
		model := collectCPUModel()
		freqMin, freqMax, freqPer := collectCPUFreq()
		temps := collectThermals()
		cpuR = cpuData{percent: percent, model: model, freqMin: freqMin, freqMax: freqMax, freqPer: freqPer, temps: temps}
	}()
	go func() {
		defer wg.Done()
		info := collectSystemInfo()
		uptime, _ := host.Uptime()
		swapT, swapU, swapF := collectSwap()
		v, _ := mem.VirtualMemory()
		sysR = sysData{
			info:   info,
			uptime: uptime,
			mem: shared.MemoryInfo{
				TotalBytes:   v.Total,
				UsedBytes:    v.Used,
				FreeBytes:    v.Available,
				UsagePercent: v.UsedPercent,
				SwapTotal:    swapT,
				SwapUsed:     swapU,
				SwapFree:     swapF,
			},
		}
	}()

	disksInfo := collectDisks()
	netInfo := collectNetwork()
	dockerInfo := collectDocker(dockerClient)

	wg.Wait()

	overallCPU := 0.0
	if len(cpuR.percent) > 0 {
		overallCPU = cpuR.percent[0]
	}

	return &shared.ReportRequest{
		Hostname:      hostname,
		System:        sysR.info,
		Timestamp:     time.Now(),
		UptimeSeconds: sysR.uptime,
		CPU: shared.CPUInfo{
			Cores:        runtime.NumCPU(),
			UsagePercent: overallCPU,
			PerCore:      cpuR.percent,
			ModelName:    cpuR.model,
			FreqMHz:      cpuR.freqPer,
			FreqMinMHz:   cpuR.freqMin,
			FreqMaxMHz:   cpuR.freqMax,
			Temperatures: cpuR.temps,
		},
		Memory:  sysR.mem,
		Disk:    disksInfo,
		Network: netInfo,
		Docker:  dockerInfo,
		GPU:     collectGPU(),
	}, nil
}

func collectDisks() []shared.DiskInfo {
	partitions, _ := disk.Partitions(true)
	seen := make(map[string]bool)
	var disksInfo []shared.DiskInfo

	for _, p := range partitions {
		if skipFstype[p.Fstype] {
			continue
		}
		if !strings.HasPrefix(p.Device, "/dev/") {
			continue
		}
		if seen[p.Device] {
			continue
		}
		seen[p.Device] = true

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		di := shared.DiskInfo{
			Device:       p.Device,
			Mount:        p.Mountpoint,
			Fstype:       p.Fstype,
			TotalBytes:   usage.Total,
			UsedBytes:    usage.Used,
			FreeBytes:    usage.Free,
			UsagePercent: usage.UsedPercent,
		}
		disksInfo = append(disksInfo, di)
	}

	ioCounters, err := disk.IOCounters()
	if err == nil {
		for i := range disksInfo {
			devName := strings.TrimPrefix(disksInfo[i].Device, "/dev/")
			io, ok := ioCounters[devName]
			if !ok {
				parent := parentRe.ReplaceAllString(devName, "")
				io, ok = ioCounters[parent]
			}
			if ok {
				disksInfo[i].ReadBytes = io.ReadBytes
				disksInfo[i].WriteBytes = io.WriteBytes
				disksInfo[i].ReadCount = io.ReadCount
				disksInfo[i].WriteCount = io.WriteCount
			}
		}
	}

	return disksInfo
}

func collectNetwork() []shared.NetworkInfo {
	netInterfaces, _ := gopsutilnet.Interfaces()
	netIOs, _ := gopsutilnet.IOCounters(false)
	ioMap := make(map[string]gopsutilnet.IOCountersStat)
	for _, io := range netIOs {
		ioMap[io.Name] = io
	}

	var netInfo []shared.NetworkInfo
	for _, iface := range netInterfaces {
		if iface.HardwareAddr == "" || strings.Contains(iface.Name, "lo") {
			continue
		}
		if skipNetPrefix(iface.Name) {
			continue
		}

		ni := shared.NetworkInfo{
			Interface: iface.Name,
			MAC:       iface.HardwareAddr,
			MTU:       iface.MTU,
		}

		var ips []string
		for _, addr := range iface.Addrs {
			ip := strings.Split(addr.Addr, "/")[0]
			if strings.Contains(ip, ".") && !strings.HasPrefix(ip, "127.") {
				ips = append(ips, ip)
			}
		}
		if len(ips) > 0 {
			ni.IP = strings.Join(ips, ", ")
		}

		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/speed", iface.Name)); err == nil {
			if speed, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				ni.SpeedMbps = speed
			}
		}

		if io, ok := ioMap[iface.Name]; ok {
			ni.BytesIn = io.BytesRecv
			ni.BytesOut = io.BytesSent
			ni.PacketsIn = io.PacketsRecv
			ni.PacketsOut = io.PacketsSent
			ni.ErrorsIn = io.Errin
			ni.ErrorsOut = io.Errout
		}

		netInfo = append(netInfo, ni)
	}

	return netInfo
}

type dockerStatsResult struct {
	index int
	stats shared.DockerContainer
	err   bool
}

// maxDockerStatsConcurrency caps how many container stats calls hit the Docker
// daemon at once, so a host with hundreds of containers does not open hundreds
// of simultaneous connections (which the daemon would throttle or refuse).
const maxDockerStatsConcurrency = 8

// dockerCallTimeout caps the whole Docker collection cycle (container list plus
// every per-container stats call). Without it a wedged daemon would block the
// stats goroutines — and therefore the receive loop and the agent's report loop
// — forever, silently halting all reporting.
const dockerCallTimeout = 3 * time.Second

// collectDockerStats enriches each container with its live stats, fetched
// concurrently but capped at maxDockerStatsConcurrency. fetch is the per-
// container stats source (nil disables enrichment) and must honor ctx so a
// hung daemon cannot stall the receive loop. Containers are updated in place
// and the same slice is returned for convenience.
func collectDockerStats(ctx context.Context, containers []shared.DockerContainer, fetch func(ctx context.Context, id string) (shared.DockerContainer, error)) []shared.DockerContainer {
	if fetch == nil || len(containers) == 0 {
		return containers
	}

	sem := make(chan struct{}, maxDockerStatsConcurrency)
	ch := make(chan dockerStatsResult, len(containers))

	for i := range containers {
		// Acquire the slot before spawning so the fan-out keeps at most
		// maxDockerStatsConcurrency goroutines alive at once, rather than
		// parking one per container on the gate. ch is buffered to len so a
		// finished goroutine never blocks on send, and the loop always drains
		// a slot — no deadlock against the receive loop below.
		sem <- struct{}{}
		go func(idx int) {
			defer func() { <-sem }()
			stats, err := fetch(ctx, containers[idx].ID)
			if err != nil {
				ch <- dockerStatsResult{index: idx, err: true}
				return
			}
			ch <- dockerStatsResult{index: idx, stats: stats}
		}(i)
	}

	for range containers {
		r := <-ch
		if r.err {
			continue
		}
		containers[r.index].CPUPercent = r.stats.CPUPercent
		containers[r.index].MemoryUsageBytes = r.stats.MemoryUsageBytes
		containers[r.index].MemoryLimitBytes = r.stats.MemoryLimitBytes
		containers[r.index].MemoryPercent = r.stats.MemoryPercent
		containers[r.index].NetworkRxBytes = r.stats.NetworkRxBytes
		containers[r.index].NetworkTxBytes = r.stats.NetworkTxBytes
	}

	return containers
}

func collectDocker(dockerClient *client.Client) shared.DockerInfo {
	if dockerClient == nil {
		return shared.DockerInfo{}
	}

	// One deadline shared across the list call and every stats call bounds the
	// entire cycle: a wedged daemon degrades to delayed/empty reports instead of
	// hanging the agent permanently.
	ctx, cancel := context.WithTimeout(context.Background(), dockerCallTimeout)
	defer cancel()

	containers, err := listContainers(ctx, dockerClient)
	if err != nil {
		return shared.DockerInfo{}
	}

	containers = collectDockerStats(ctx, containers, func(cctx context.Context, id string) (shared.DockerContainer, error) {
		return getContainerStats(cctx, dockerClient, id)
	})
	return shared.DockerInfo{Containers: containers}
}

func collectSystemInfo() shared.SystemInfo {
	info := shared.SystemInfo{}

	osReleasePaths := []string{"/host/etc/os-release", "/etc/os-release"}
	if root := hostRoot(); root != "" {
		// Prefer the real host's os-release over the container's base image.
		osReleasePaths = append([]string{filepath.Join(root, "etc", "os-release")}, osReleasePaths...)
	}

	for _, path := range osReleasePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			val := strings.Trim(parts[1], "\"")
			switch parts[0] {
			case "ID":
				if info.OS == "" {
					info.OS = val
				}
			case "NAME":
				if info.Platform == "" {
					info.Platform = val
				}
			case "VERSION_ID":
				if info.PlatformVer == "" {
					info.PlatformVer = val
				}
			}
		}
		if info.OS != "" {
			break
		}
	}

	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		info.Kernel = strings.TrimSpace(string(data))
	} else if h, err := host.Info(); err == nil && h.KernelVersion != "" {
		info.Kernel = h.KernelVersion
	}

	info.Arch = runtime.GOARCH

	if h, err := host.Info(); err == nil {
		info.VirtSystem = h.VirtualizationSystem
		info.VirtRole = h.VirtualizationRole
	}

	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 4 {
			info.Load1, _ = strconv.ParseFloat(fields[0], 64)
			info.Load5, _ = strconv.ParseFloat(fields[1], 64)
			info.Load15, _ = strconv.ParseFloat(fields[2], 64)
			procs := strings.Split(fields[3], "/")
			if len(procs) == 2 {
				info.ProcessCount, _ = strconv.Atoi(procs[1])
			}
		}
	}

	return info
}
