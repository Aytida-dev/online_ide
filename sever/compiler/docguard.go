package compiler

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"time"

	"github.com/docker/docker/api/types/container"
)

var idleContainers = map[string]string{}

func (dm *DockerManager) MonitorResources() {
	time.Sleep(MONITORING_INTERVAL)

	if idleContainers == nil {
		idleContainers = make(map[string]string)
	}
	if dm.containerResources == nil {
		dm.containerResources = make(map[string]ContainerResources)
	}
	if dm.reusableContainers == nil {
		dm.reusableContainers = make(map[string]map[string]int)
	}
	if dm.filledContainers == nil {
		dm.filledContainers = make(map[string]map[string]int)
	}

	for {
		select {
		case <-dm.ctx.Done():
			return
		default:
			dm.checkAndUpdateResources()
		}

		time.Sleep(MONITORING_INTERVAL)
	}
}

func (dm *DockerManager) checkAndUpdateResources() {
	if len(dm.containerResources) == 0 {
		return
	}

	containerResources := make(map[string]ContainerResources)
	resuableContainers := make(map[string]map[string]int)
	filledContainers := make(map[string]map[string]int)

	containersToRemove := make(map[string]string)
	containersToupdate := make(map[string]ContainerResources)

	log.Print("lock by checkAndUpdateResources copy")
	start := time.Now()
	dm.mu.Lock()

	maps.Copy(resuableContainers, dm.reusableContainers)
	maps.Copy(filledContainers, dm.filledContainers)
	maps.Copy(containerResources, dm.containerResources)

	dm.mu.Unlock()
	log.Print("unlock by checkAndUpdateResources copy: ", time.Since(start))

	for containerID := range containerResources {
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

		stats, err := dm.getContainerStats(containerID)
		if err != nil {
			log.Printf("Failed to get stats for container %s: %v", containerID, err)
			containersToRemove[containerID] = lang
			continue
		}

		resources := containerResources[containerID]

		var newMem int64
		var newCpu int64
		var toChange bool

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

		if stats.memoryPercentage < float64(opts.MemIdleThreshold) && stats.cpuPercentage < float64(opts.CpuIdleThreshold) {
			toChange = false
			if _, ok := idleContainers[containerID]; !ok {
				containersToRemove[containerID] = lang
			} else {
				idleContainers[containerID] = lang
			}
		}

		if toChange {
			log.Printf("Scaling container %s memory from %d MB to %d MB, CPU from %d to %d",
				containerID, resources.CurrentMemory/(1024*1024), newMem/(1024*1024),
				resources.CurrentCPU, newCpu)

			err := dm.updateContainerResources(containerID, newMem, newCpu)
			if err != nil {
				log.Printf("Failed to update resources for container %s: %v", containerID, err)
				containersToRemove[containerID] = lang
				continue
			} else {
				resources.CurrentMemory = newMem
				resources.CurrentCPU = newCpu

				containersToupdate[containerID] = resources
			}
		}

	}

	log.Print("lock by checkAndUpdateResources remove")
	start1 := time.Now()
	dm.mu.Lock()
	defer func() {
		log.Print("unlock by checkAndUpdateResources remove: ", time.Since(start1))
		dm.mu.Unlock()
	}()

	maps.Copy(dm.containerResources, containersToupdate)

	for containerID, lang := range containersToRemove {
		delete(dm.reusableContainers[lang], containerID)
		delete(dm.filledContainers[lang], containerID)
		delete(dm.containerResources, containerID)

		err := dm.RemoveContainer(containerID, lang)
		if err != nil {
			log.Printf("Failed to remove container %s: %v", containerID, err)
		}
		log.Print("Idle Container or stucked container removed : ", containerID)
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
