package compiler

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/mount"
)

var LangImages = map[string]LangOptions{
	"js": {
		Image:      "node:22.14-alpine",
		IsCompiled: false,
		ExecCmd:    func(s string) []string { return []string{"node", "-e", s} },
		CompileCmd: nil,
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeVolume,
				Source:   "vol-npm",
				Target:   "/usr/local/lib/node_modules",
				ReadOnly: true,
			},
		},
		RunOnHost:      nil,
		MinCpu:         1,
		MinMem:         128 * 1024 * 1024,
		IncrementalMem: 100 * 1024 * 1024,
		IncrementalCpu: 1,
		MaxMem:         1024 * 1024 * 1024,
		MaxCpu:         2,
		Env: []string{
			"HOME=/tmp",
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		CpuIdleThreshold: 5,
		MemIdleThreshold: 15,
	},
	"ts": {
		Image:      "node:22.14-alpine",
		IsCompiled: true,
		ExecCmd:    func(s string) []string { return []string{"node", s} },
		CompileCmd: nil,
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeVolume,
				Source:   "vol-npm",
				Target:   "/usr/local/lib/node_modules",
				ReadOnly: true,
			},
			{
				Type:     mount.TypeBind,
				Source:   COMPILED_FILES,
				Target:   CONTAINER_COMPILED_FILES,
				ReadOnly: true,
			},
		},
		RunOnHost:      func(file string) []string { return []string{"tsc", file, "-outDir", COMPILED_FILES} },
		FileName:       func(cont string) string { return fmt.Sprintf("%s-%d-code.ts", cont, time.Now().UnixNano()) },
		MinCpu:         1,
		MinMem:         128 * 1024 * 1024,
		IncrementalMem: 100 * 1024 * 1024,
		IncrementalCpu: 1,
		MaxMem:         1024 * 1024 * 1024,
		MaxCpu:         2,
		Env: []string{
			"HOME=/tmp",
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		CpuIdleThreshold: 5,
		MemIdleThreshold: 15,
	},
	"py": {
		Image:      "python:3.12-alpine",
		IsCompiled: false,
		ExecCmd: func(s string) []string {
			return []string{"python3", "-c", s}
		},
		CompileCmd: nil,
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeVolume,
				Source:   "vol-pip",
				Target:   "/opt/py-packages",
				ReadOnly: true,
			},
		},
		RunOnHost:      nil,
		MinCpu:         1,
		MinMem:         128 * 1024 * 1024,
		IncrementalMem: 100 * 1024 * 1024,
		IncrementalCpu: 1,
		MaxMem:         1024 * 1024 * 1024,
		MaxCpu:         2,
		Env: []string{
			"HOME=/tmp",
			"PYTHONUNBUFFERED=1",
			"PYTHONPATH=/opt/py-packages",
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		CpuIdleThreshold: 5,
		MemIdleThreshold: 15,
	},
	"py-ml": {
		Image:      "python:3.12-alpine",
		IsCompiled: false,
		ExecCmd: func(s string) []string {
			return []string{"python3", "-c", s}
		},
		CompileCmd: nil,
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeVolume,
				Source:   "vol-pip",
				Target:   "/opt/py-packages", // pip --target
				ReadOnly: true,
			},
		},
		RunOnHost:      nil,
		MinCpu:         2,
		MinMem:         256 * 1024 * 1024,
		IncrementalMem: 100 * 1024 * 1024,
		IncrementalCpu: 1,
		MaxMem:         1024 * 1024 * 1024,
		MaxCpu:         4,
		Env: []string{
			"HOME=/tmp",
			"PYTHONUNBUFFERED=1",
			"PYTHONPATH=/opt/py-packages",
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		CpuIdleThreshold: 5,
		MemIdleThreshold: 30,
	},
	"c": {
		Image:      "debian:12.10-slim",
		IsCompiled: true,
		ExecCmd:    func(s string) []string { return []string{s} },
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   COMPILED_FILES,
				Target:   CONTAINER_COMPILED_FILES,
				ReadOnly: true,
			},
		},
		RunOnHost: func(file string) []string {
			return []string{"gcc", file, "-o", COMPILED_FILES + "/" + strings.TrimSuffix(filepath.Base(file), ".c") + ".out"}
		},
		FileName: func(containerID string) string {
			return fmt.Sprintf("%s-%d-code.c", containerID, time.Now().UnixNano())
		},
		MinCpu:           1,
		MinMem:           128 * 1024 * 1024,
		IncrementalMem:   100 * 1024 * 1024,
		IncrementalCpu:   1,
		MaxMem:           1024 * 1024 * 1024,
		MaxCpu:           2,
		CpuIdleThreshold: 3,
		MemIdleThreshold: 5,
	},
	"cpp": {
		Image:      "gcc:14",
		IsCompiled: true,
		ExecCmd:    func(s string) []string { return []string{s} },
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   COMPILED_FILES,
				Target:   CONTAINER_COMPILED_FILES,
				ReadOnly: true,
			},
		},
		RunOnHost: func(file string) []string {
			return []string{"g++", file, "-o", COMPILED_FILES + "/" + strings.TrimSuffix(filepath.Base(file), ".cpp") + ".out"}
		},
		FileName: func(containerID string) string {
			return fmt.Sprintf("%s-%d-code.cpp", containerID, time.Now().UnixNano())
		},
		MinCpu:           1,
		MinMem:           128 * 1024 * 1024,
		IncrementalMem:   100 * 1024 * 1024,
		IncrementalCpu:   1,
		MaxMem:           1024 * 1024 * 1024,
		MaxCpu:           2,
		CpuIdleThreshold: 3,
		MemIdleThreshold: 5,
	},
	"java": {
		Image:      "openjdk:21-slim",
		IsCompiled: true,
		ExecCmd: func(s string) []string { // s = /tmp/tmp_compiled/<user>
			files, err := os.ReadDir(s)
			if err != nil {
				return []string{"java", "-cp", s, "Main"}
			}
			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".class") {
					return []string{"java", "-cp", s, strings.TrimSuffix(file.Name(), ".class")}
				}
			}
			return []string{"java", "-cp", s, "Main"}
		},
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   COMPILED_FILES,
				Target:   CONTAINER_COMPILED_FILES,
				ReadOnly: true,
			},
		},
		RunOnHost: func(file string) []string {
			id := time.Now().UnixNano()
			userId := fmt.Sprintf("%d-%d", id, len(file))
			compiledDir := filepath.Join(COMPILED_FILES, userId)
			_ = os.MkdirAll(compiledDir, 0755)
			return []string{"javac", "-d", compiledDir, file}
		},
		FileName: func(containerID string) string {
			return fmt.Sprintf("%s-%d-code.java", containerID, time.Now().UnixNano())
		},
		MinCpu:         1,
		MinMem:         256 * 1024 * 1024,
		IncrementalMem: 128 * 1024 * 1024,
		IncrementalCpu: 1,
		MaxMem:         1024 * 1024 * 1024,
		MaxCpu:         2,
		Env: []string{
			"HOME=/tmp",
			"PATH=/usr/local/openjdk-21/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		MemIdleThreshold: 15,
		CpuIdleThreshold: 3,
	},
	"php": {
		Image:      "php:8.3-cli",
		IsCompiled: false,
		ExecCmd: func(s string) []string {
			fileName := fmt.Sprintf("%s-%d-code.php", time.Now().Format("2006-01-02_15-04-05"), time.Now().UnixNano())
			if err := os.WriteFile(fileName, []byte(s), 0644); err != nil {
				log.Printf("failed to write file: %v", err)
				return []string{"php", "-r", s}

			}
			return []string{"php", fileName}
		},
		RunOnHost:      nil,
		FileName:       nil,
		MinCpu:         1,
		MinMem:         64 * 1024 * 1024,
		IncrementalMem: 64 * 1024 * 1024,
		IncrementalCpu: 1,
		MaxMem:         256 * 1024 * 1024,
		MaxCpu:         1,
		Env: []string{
			"HOME=/tmp",
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		}, MemIdleThreshold: 5,
		CpuIdleThreshold: 3,
	},
}
