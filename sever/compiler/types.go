package compiler

import (
	"context"
	"sync"
	"time"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

const (
	MAX_USERS                   = 2
	MONITORING_INTERVAL         = 5 * time.Minute
	MEMORY_USAGE_HIGH_THRESHOLD = 90
	MEMORY_USAGE_LOW_THRESHOLD  = 30
	CPU_UNIT                    = 50_000 // 1/2 core
	COMPILED_FILES              = "/tmp/tmp_compiled"
	CODE_FILES_DIR              = "/tmp/code_files"
	CONTAINER_COMPILED_FILES    = "/tmp/tmp_compiled"
)

type LangOptions struct {
	Image            string
	IsCompiled       bool
	ExecCmd          func(string) []string
	CompileCmd       func(string) []string
	MinCpu           int64
	MinMem           int64
	IncrementalMem   int64
	IncrementalCpu   int64
	MaxMem           int64
	MaxCpu           int64
	Mounts           []mount.Mount
	Env              []string
	RunOnHost        func(string) []string
	FileName         func(string) string
	CpuIdleThreshold int64
	MemIdleThreshold int64
}

type ContainerResources struct {
	CurrentMemory int64
	CurrentCPU    int64
}

type DockerManager struct {
	cli                *client.Client
	mu                 sync.Mutex
	reusableContainers map[string]map[string]int
	filledContainers   map[string]map[string]int
	runningContainers  map[string]int
	containerResources map[string]ContainerResources
	ctx                context.Context
	cancel             context.CancelFunc
}

type containerStats struct {
	memoryUsage      uint64
	memoryLimit      uint64
	memoryPercentage float64
	cpuPercentage    float64
}
