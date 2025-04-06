package compiler

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"time"

	"github.com/docker/docker/api/types/container"
)

func (dm *DockerManager) MonitorResources() {
	ticker := time.NewTicker(MONITORING_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-ticker.C:
			dm.checkAndUpdateResources()
		}
	}
}

func (dm *DockerManager) checkAndUpdateResources() {
	// create a hardcopy of the dm

	if len(dm.containerResources) == 0 {
		return
	}

	containerResources := make(map[string]ContainerResources)
	resuableContainers := make(map[string]map[string]int)
	filledContainers := make(map[string]map[string]int)

	dm.mu.Lock()

	maps.Copy(resuableContainers, dm.reusableContainers)
	maps.Copy(filledContainers, dm.filledContainers)
	maps.Copy(containerResources, dm.containerResources)

	dm.mu.Unlock()

	startTime := time.Now()
	defer func() {
		log.Printf("Resource check took %s", time.Since(startTime))
	}()

	for containerID := range containerResources {
		// Get container stats
		stats, err := dm.getContainerStats(containerID)
		if err != nil {
			log.Printf("Failed to get stats for container %s: %v", containerID, err)
			continue
		}

		// log.Print("Container stats: ", stats, " for container: ", containerID)

		// Find language for this container
		var lang string
		var opts LangOptions
		for l, containers := range resuableContainers {
			if _, ok := containers[containerID]; ok {
				lang = l
				opts = LangImages[l]
				break
			}
		}
		if lang == "" {
			for l, containers := range filledContainers {
				if _, ok := containers[containerID]; ok {
					lang = l
					opts = LangImages[l]
					break
				}
			}
		}
		if lang == "" {
			log.Printf("Cannot find language for container %s", containerID)
			continue
		}

		resources := containerResources[containerID]

		var newMem int64
		var newCpu int64
		var toChange bool

		// Check if memory usage is high
		if stats.memoryPercentage > MEMORY_USAGE_HIGH_THRESHOLD && resources.CurrentMemory < opts.MaxMem {
			newMem = min(resources.CurrentMemory+opts.IncrementalMem, opts.MaxMem)
			toChange = true

		} else if stats.memoryPercentage < MEMORY_USAGE_LOW_THRESHOLD && resources.CurrentMemory > opts.MinMem {
			newMem = max(resources.CurrentMemory-opts.IncrementalMem, opts.MinMem)
			toChange = true
		}

		if stats.cpuPercentage > MEMORY_USAGE_HIGH_THRESHOLD && resources.CurrentCPU < opts.MaxCpu {
			newCpu = min(resources.CurrentCPU+opts.IncrementalCpu, opts.MaxCpu)
			toChange = true
		} else if stats.cpuPercentage < MEMORY_USAGE_LOW_THRESHOLD && resources.CurrentCPU > opts.MinCpu {
			newCpu = max(resources.CurrentCPU-opts.IncrementalCpu, opts.MinCpu)
			toChange = true
		}

		if toChange {
			log.Printf("Scaling container %s memory from %d MB to %d MB, CPU from %d to %d",
				containerID, resources.CurrentMemory/(1024*1024), newMem/(1024*1024),
				resources.CurrentCPU, newCpu)

			err := dm.updateContainerResources(containerID, newMem, newCpu)
			if err != nil {
				log.Printf("Failed to update resources for container %s: %v", containerID, err)
			} else {
				resources.CurrentMemory = newMem
				resources.CurrentCPU = newCpu

				dm.mu.Lock()
				dm.containerResources[containerID] = resources
				dm.mu.Unlock()
			}
		}

	}
}

func (dm *DockerManager) getContainerStats(containerID string) (containerStats, error) {
	stats := containerStats{}

	resp, err := dm.cli.ContainerStats(dm.ctx, containerID, false)
	if err != nil {
		return stats, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer resp.Body.Close()

	var statsJSON container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&statsJSON); err != nil {
		return stats, fmt.Errorf("failed to decode stats: %w", err)
	}

	stats.memoryUsage = statsJSON.MemoryStats.Usage
	stats.memoryLimit = statsJSON.MemoryStats.Limit
	stats.memoryPercentage = float64(stats.memoryUsage) / float64(stats.memoryLimit) * 100.0

	cpuDelta := float64(statsJSON.CPUStats.CPUUsage.TotalUsage - statsJSON.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(statsJSON.CPUStats.SystemUsage - statsJSON.PreCPUStats.SystemUsage)
	if systemDelta > 0 && cpuDelta > 0 {
		stats.cpuPercentage = (cpuDelta / systemDelta) * float64(len(statsJSON.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}

	return stats, nil
}

func (dm *DockerManager) updateContainerResources(containerID string, memory int64, cpu int64) error {
	updateConfig := container.UpdateConfig{
		Resources: container.Resources{
			Memory:     memory,
			MemorySwap: memory * 2,
			CPUPeriod:  100000,
			CPUQuota:   int64(cpu * CPU_UNIT),
		},
	}

	_, err := dm.cli.ContainerUpdate(dm.ctx, containerID, updateConfig)
	return err
}
