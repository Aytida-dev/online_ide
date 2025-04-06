package compiler

import (
	"fmt"
	"log"
)

func (dm *DockerManager) FindContainer(lang string) (string, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	var bestContainerID string
	if containers, ok := dm.reusableContainers[lang]; ok && len(containers) > 0 {
		for k, v := range containers {
			if bestContainerID == "" || v < containers[bestContainerID] {
				bestContainerID = k
			}
		}

	}

	if bestContainerID == "" {
		id, err := dm.CreateContainer(lang)
		if err != nil {
			return "", fmt.Errorf("failed to create container: %w", err)
		}
		log.Print("Creating new container because no best found: ", id)
		dm.reusableContainers[lang][id] = 1

		return id, nil
	}

	if dm.reusableContainers[lang][bestContainerID] >= MAX_USERS {
		id, err := dm.CreateContainer(lang)
		if err != nil {
			return "", fmt.Errorf("failed to create container: %w", err)
		}
		log.Print("Creating new container because best is full: ", id)

		dm.reusableContainers[lang][id] = 1

		return id, nil
	}

	currentUsage := dm.reusableContainers[lang][bestContainerID]

	if currentUsage == MAX_USERS-1 {
		delete(dm.reusableContainers[lang], bestContainerID)

		if dm.filledContainers[lang] == nil {
			dm.filledContainers[lang] = make(map[string]int)
		}

		dm.filledContainers[lang][bestContainerID] = currentUsage + 1

	} else {
		dm.reusableContainers[lang][bestContainerID] = currentUsage + 1
	}

	log.Print("Reusing container: ", bestContainerID)

	return bestContainerID, nil
}

func (dm *DockerManager) DecreaseUser(containerID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	for lang, containers := range dm.reusableContainers {
		if _, ok := containers[containerID]; ok {
			if containers[containerID] > 1 {
				containers[containerID]--
				log.Print("Decreasing user count for container: resuable ", containerID)
			} else {
				delete(containers, containerID)
				if err := dm.RemoveContainer(containerID, lang); err != nil {
					return fmt.Errorf("failed to remove container: %w", err)
				}
			}
		}
	}

	for lang, containers := range dm.filledContainers {
		if _, ok := containers[containerID]; ok {
			containers[containerID]--
			log.Print("Decreasing user count for container: filled ", containerID)
			if containers[containerID] < MAX_USERS {
				delete(containers, containerID)
				dm.reusableContainers[lang][containerID] = MAX_USERS - 1
			}
		}
	}

	return nil
}
