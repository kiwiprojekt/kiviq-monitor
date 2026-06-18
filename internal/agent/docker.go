package agent

import (
	"context"
	"encoding/json"

	"github.com/michal/kiviq/internal/shared"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

func listContainers(ctx context.Context, dockerClient *client.Client) ([]shared.DockerContainer, error) {
	result, err := dockerClient.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var containers []shared.DockerContainer
	for _, c := range result.Items {
		containers = append(containers, summaryToContainer(c))
	}
	return containers, nil
}

// summaryToContainer maps a Docker list-API summary to our DockerContainer.
// Summary.State is already a populated container state, so no per-container
// ContainerInspect round-trip is needed to read it.
func summaryToContainer(c container.Summary) shared.DockerContainer {
	name := ""
	if len(c.Names) > 0 {
		name = c.Names[0]
	}

	id := c.ID
	if len(id) > 12 {
		id = id[:12]
	}

	var ports []shared.DockerPort
	for _, p := range c.Ports {
		ports = append(ports, shared.DockerPort{
			IP:          p.IP.String(),
			PrivatePort: p.PrivatePort,
			PublicPort:  p.PublicPort,
			Type:        p.Type,
		})
	}

	return shared.DockerContainer{
		ID:     id,
		Name:   name,
		Image:  c.Image,
		Status: c.Status,
		State:  string(c.State),
		Ports:  ports,
	}
}

func getContainerStats(ctx context.Context, dockerClient *client.Client, containerID string) (shared.DockerContainer, error) {
	statsResult, err := dockerClient.ContainerStats(ctx, containerID, client.ContainerStatsOptions{
		Stream:                false,
		IncludePreviousSample: true,
	})
	if err != nil {
		return shared.DockerContainer{}, err
	}
	defer statsResult.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(statsResult.Body).Decode(&stats); err != nil {
		return shared.DockerContainer{}, err
	}

	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	numCPU := float64(stats.CPUStats.OnlineCPUs)
	if numCPU == 0 {
		numCPU = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	cpuPercent := 0.0
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * numCPU * 100
	}

	memUsage := stats.MemoryStats.Usage
	memLimit := stats.MemoryStats.Limit
	memPercent := 0.0
	if memLimit > 0 {
		memPercent = float64(memUsage) / float64(memLimit) * 100
	}

	var rxBytes, txBytes uint64
	for _, network := range stats.Networks {
		rxBytes += network.RxBytes
		txBytes += network.TxBytes
	}

	return shared.DockerContainer{
		CPUPercent:       cpuPercent,
		MemoryUsageBytes: memUsage,
		MemoryLimitBytes: memLimit,
		MemoryPercent:    memPercent,
		NetworkRxBytes:   rxBytes,
		NetworkTxBytes:   txBytes,
	}, nil
}
