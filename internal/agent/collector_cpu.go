package agent

import (
	"os"
	"strconv"
	"strings"
)

func collectCPUModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func collectCPUFreq() (minMHz float64, maxMHz float64, perCore []float64) {
	perCore = readCPUScalingFreqs()
	if len(perCore) == 0 {
		return 0, 0, nil
	}

	basePath := "/sys/devices/system/cpu/cpu0/cpufreq"
	if data, err := os.ReadFile(basePath + "/scaling_min_freq"); err == nil {
		if v, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			minMHz = v / 1000
		}
	}
	if data, err := os.ReadFile(basePath + "/scaling_max_freq"); err == nil {
		if v, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			maxMHz = v / 1000
		}
	}

	return minMHz, maxMHz, perCore
}

func readCPUScalingFreqs() []float64 {
	var freqs []float64
	for i := 0; ; i++ {
		path := "/sys/devices/system/cpu/cpu" + strconv.Itoa(i) + "/cpufreq/scaling_cur_freq"
		data, err := os.ReadFile(path)
		if err != nil {
			break
		}
		val, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		if err != nil {
			continue
		}
		freqs = append(freqs, val/1000)
	}
	return freqs
}
