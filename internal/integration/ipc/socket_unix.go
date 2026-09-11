//go:build !windows

package ipc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func ipcRuntimeDir() (string, error) {
	if base := os.Getenv("XDG_RUNTIME_DIR"); filepath.IsAbs(base) {
		if err := validatePrivateRuntimeDir(base); err != nil {
			return "", fmt.Errorf("validate XDG_RUNTIME_DIR: %w", err)
		}
		return filepath.Join(base, "tmus"), nil
	}

	base := os.TempDir()
	if !filepath.IsAbs(base) {
		return "", fmt.Errorf("temporary directory is not absolute: %s", base)
	}
	return filepath.Join(base, fmt.Sprintf("tmus-%d", os.Getuid())), nil
}

// prepareRuntimeDir creates and validates the directory that contains tmus IPC
// resources. Refusing symlinks, foreign ownership, and group or other access
// keeps the socket private from the moment it is created.
func prepareRuntimeDir(dir string) error {
	created := false
	if err := os.Mkdir(dir, 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create IPC directory: %w", err)
		}
	} else {
		created = true
	}
	if created {
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("secure IPC directory: %w", err)
		}
	}
	if err := validatePrivateRuntimeDir(dir); err != nil {
		return fmt.Errorf("validate IPC directory: %w", err)
	}
	return nil
}

func validatePrivateRuntimeDir(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect runtime directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("runtime directory must not be a symbolic link: %s", dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("runtime path is not a directory: %s", dir)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("inspect runtime directory ownership: %s", dir)
	}
	if int(stat.Uid) != os.Getuid() {
		return fmt.Errorf("runtime directory is not owned by the current user: %s", dir)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		return fmt.Errorf("runtime directory must have mode 0700, got %04o: %s", perm, dir)
	}
	return nil
}

func secureUnixSocket(socketPath string) error {
	return os.Chmod(socketPath, 0o600)
}

func validateSocketPath(_ string) error {
	return nil
}

func isUnixSocketAddrInUse(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE)
}

func isUnixSocketUnsupported(_ error) bool {
	return false
}

func isNoServer(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	return errors.Is(err, syscall.ECONNREFUSED)
}
