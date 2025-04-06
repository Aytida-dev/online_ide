package compiler

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
)

func NewDockerManager() (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	for _, opts := range LangImages {
		_, err := cli.ImageInspect(ctx, opts.Image)
		if err != nil {
			if _, err := cli.ImagePull(ctx, opts.Image, image.PullOptions{}); err != nil {
				cancel()
				return nil, fmt.Errorf("failed to pull image: %w", err)
			}
		}

		for _, m := range opts.Mounts {
			if m.Type == mount.TypeVolume {
				if _, err := cli.VolumeCreate(ctx, volume.CreateOptions{
					Name:   m.Source,
					Driver: "local",
				}); err != nil {
					cancel()
					return nil, fmt.Errorf("failed to create volume: %w", err)
				}
			}
		}

		if _, err := os.Stat(CODE_FILES_DIR); os.IsNotExist(err) {
			if err := os.MkdirAll(CODE_FILES_DIR, 0755); err != nil {
				cancel()
				return nil, fmt.Errorf("failed to create directory: %w", err)
			}
		}
		if _, err := os.Stat(COMPILED_FILES); os.IsNotExist(err) {
			if err := os.MkdirAll(COMPILED_FILES, 0755); err != nil {
				cancel()
				return nil, fmt.Errorf("failed to create directory: %w", err)
			}
		}

	}
	return &DockerManager{
		cli:                cli,
		reusableContainers: make(map[string]map[string]int),
		filledContainers:   make(map[string]map[string]int),
		runningContainers:  map[string]int{},
		containerResources: make(map[string]ContainerResources),
		ctx:                ctx,
		cancel:             cancel,
	}, nil
}

func (dm *DockerManager) CreateContainer(lang string) (string, error) {
	ctx := context.Background()
	opt, ok := LangImages[lang]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", lang)
	}

	// mounts := []mount.Mount{}

	// mounts = append(mounts, opt.Mounts...)

	testTimeout := 60 * 5

	config := &container.Config{
		Image:        opt.Image,
		Tty:          true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"sh"},
		StopTimeout:  &testTimeout,
		Env:          opt.Env,
	}

	hostConfig := &container.HostConfig{
		AutoRemove:     false,
		SecurityOpt:    []string{"no-new-privileges"},
		CapDrop:        []string{"ALL"},
		ReadonlyRootfs: true,

		Resources: container.Resources{
			Memory:      int64(opt.MinMem),
			CPUPeriod:   100000,
			CPUQuota:    opt.MinCpu * CPU_UNIT,
			MemorySwap:  opt.MinMem * 2,
			CPUShares:   512,
			BlkioWeight: 100,
			PidsLimit:   func(i int64) *int64 { return &i }(100),
			Ulimits: []*units.Ulimit{
				{
					Name: "nproc",
					Hard: 100,
					Soft: 50,
				},
				{
					Name: "nofile",
					Hard: 100,
					Soft: 50,
				},
			},
		},

		Mounts: opt.Mounts,
		RestartPolicy: container.RestartPolicy{
			Name:              "always",
			MaximumRetryCount: 0,
		},
	}

	resp, err := dm.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := dm.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	dm.runningContainers[lang]++
	if dm.reusableContainers[lang] == nil {
		dm.reusableContainers[lang] = make(map[string]int)
	}
	if dm.filledContainers[lang] == nil {
		dm.filledContainers[lang] = make(map[string]int)
	}

	dm.containerResources[resp.ID] = ContainerResources{
		CurrentMemory: opt.MinMem,
		CurrentCPU:    opt.MinCpu,
	}

	return resp.ID, nil
}

func (dm *DockerManager) RemoveContainer(containerID string, lang string) error {
	ctx := context.Background()

	dm.runningContainers[lang]--
	if dm.runningContainers[lang] == 0 {
		delete(dm.runningContainers, lang)
	}
	if dm.reusableContainers[lang] == nil {
		dm.reusableContainers[lang] = make(map[string]int)
	}
	if dm.filledContainers[lang] == nil {
		dm.filledContainers[lang] = make(map[string]int)
	}

	delete(dm.reusableContainers[lang], containerID)
	delete(dm.filledContainers[lang], containerID)

	delete(dm.containerResources, containerID)

	log.Print("Removing container: ", containerID)

	return dm.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

func (dm *DockerManager) Shutdown() {
	dm.cancel()
}
