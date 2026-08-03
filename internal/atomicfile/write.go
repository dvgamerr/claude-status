// Package atomicfile writes complete files without exposing partial content.
package atomicfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Write replaces path with data after flushing both the file and its parent
// directory. The parent directory must already exist.
func Write(path string, data []byte, mode os.FileMode) (returnErr error) {
	if strings.TrimSpace(path) == "" {
		return errors.New("destination path is empty")
	}
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".atomic-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tempName := temp.Name()
	closed := false
	defer func() {
		if !closed {
			_ = temp.Close()
		}
		if returnErr != nil {
			_ = os.Remove(tempName)
		}
	}()

	if err := temp.Chmod(mode); err != nil {
		return fmt.Errorf("set temporary file mode: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		closed = true
		return fmt.Errorf("close temporary file: %w", err)
	}
	closed = true

	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace %q: %w", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("set mode on %q: %w", path, err)
	}
	if err := syncDir(dir); err != nil {
		return fmt.Errorf("sync parent directory: %w", err)
	}
	return nil
}
