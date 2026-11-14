package privatefile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writePrivateFile(path string, content []byte) (returnErr error) {
	return writePrivateFileWithReplace(path, content, replacePrivateFile)
}

func writePrivateFileWithReplace(path string, content []byte, replace func(string, string) error) (returnErr error) {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create private directory: %w", err)
	}
	if err := rejectSymlinkPath(directory); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".runtime-env-*.tmp")
	if err != nil {
		return fmt.Errorf("create private temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	// Protect the empty temporary file before any secret bytes are written.
	if err := protectPrivateFile(temporaryPath); err != nil {
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		return fmt.Errorf("write private temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync private temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close private temporary file: %w", err)
	}
	if err := rejectSymlinkPath(directory); err != nil {
		return err
	}
	if _, err := os.Lstat(path); err == nil {
		if err := rejectSymlinkPath(path); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect private runtime file: %w", err)
	}
	if err := replace(temporaryPath, path); err != nil {
		return fmt.Errorf("install private runtime file: %w", err)
	}
	return nil
}

func rejectSymlinkPath(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return errors.New("runtime file path is invalid")
	}
	current := filepath.VolumeName(absolute) + string(filepath.Separator)
	remainder := strings.TrimPrefix(absolute, current)
	for _, part := range strings.Split(remainder, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return fmt.Errorf("inspect runtime file path: %w", statErr)
		}
		unsafe, unsafeErr := isUnsafePathComponent(current, info)
		if unsafeErr != nil {
			return fmt.Errorf("inspect runtime file path safety: %w", unsafeErr)
		}
		if unsafe {
			return errors.New("runtime file path must not contain symlinks or reparse points")
		}
	}
	return nil
}

func Write(path string, data []byte) error { return writePrivateFile(path, data) }
func WriteWithReplace(path string, data []byte, replace func(string, string) error) error {
	return writePrivateFileWithReplace(path, data, replace)
}
func RejectPath(path string) error { return rejectSymlinkPath(path) }
