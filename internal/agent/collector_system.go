package agent

import (
	"os"
	"strconv"
	"strings"
)

func collectSwap() (total, used, free uint64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}

	values := make(map[string]uint64)
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		valStr = strings.TrimSuffix(valStr, " kB")

		val, err := strconv.ParseUint(valStr, 10, 64)
		if err != nil {
			continue
		}
		values[key] = val * 1024
	}

	total = values["SwapTotal"]
	free = values["SwapFree"]
	if total > free {
		used = total - free
	}

	return total, used, free
}
