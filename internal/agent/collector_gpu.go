package agent

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/michal/kiviq/internal/shared"
)

func collectGPU() shared.GPUInfo {
	path, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return shared.GPUInfo{}
	}

	cmd := exec.Command(path,
		"--query-gpu=name,utilization.gpu,memory.total,memory.used,temperature.gpu,power.draw,driver_version",
		"--format=csv,noheader,nounits",
	)

	output, err := cmd.Output()
	if err != nil {
		return shared.GPUInfo{}
	}

	return parseGPUOutput(string(output))
}

func parseGPUOutput(output string) shared.GPUInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return shared.GPUInfo{}
	}

	fields := strings.Split(lines[0], ", ")
	if len(fields) < 7 {
		return shared.GPUInfo{}
	}

	gpu := shared.GPUInfo{
		Name:   strings.TrimSpace(fields[0]),
		Driver: strings.TrimSpace(fields[6]),
	}

	gpu.Utilization, _ = strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)

	if v, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64); err == nil {
		gpu.MemoryTotal = uint64(v) * 1024 * 1024
	}

	if v, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64); err == nil {
		gpu.MemoryUsed = uint64(v) * 1024 * 1024
	}

	gpu.Temperature, _ = strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
	gpu.PowerDraw, _ = strconv.ParseFloat(strings.TrimSpace(fields[5]), 64)

	return gpu
}
