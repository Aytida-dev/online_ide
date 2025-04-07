package compiler

import (
	"os"
	"path/filepath"
)

func removeContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			err = os.RemoveAll(entryPath)
		} else {
			err = os.Remove(entryPath)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (dm *DockerManager) CleanupCompiledFiles() error {
	return removeContents(COMPILED_FILES)
}

func (dm *DockerManager) CleanupCodeFiles() error {
	return removeContents(CODE_FILES_DIR)
}
