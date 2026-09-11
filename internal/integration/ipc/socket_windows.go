//go:build windows

package ipc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// Windows reserves one byte of sockaddr_un.Path for the terminating NUL.
const maxWindowsSocketPathBytes = 107

func ipcRuntimeDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve local application data directory: %w", err)
	}
	if !filepath.IsAbs(base) {
		return "", fmt.Errorf("local application data directory is not absolute: %s", base)
	}
	return filepath.Join(base, "tmus", "run"), nil
}

// prepareRuntimeDir validates the existing LocalAppData base, then creates and
// validates tmus/run one component at a time. Existing permissions are never
// rewritten. Handles deny deletion until all components have been checked.
func prepareRuntimeDir(dir string) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return fmt.Errorf("identify IPC user: %w", err)
	}
	sd, err := windowsRuntimeSecurity(user.User.Sid)
	if err != nil {
		return fmt.Errorf("build IPC directory permissions: %w", err)
	}
	appDir := filepath.Dir(dir)
	base := filepath.Dir(appDir)
	for _, path := range []string{base, appDir, dir} {
		if path != base {
			if err := createWindowsRuntimeDir(path, sd); err != nil {
				return fmt.Errorf("create IPC directory %s: %w", path, err)
			}
		}
		h, err := openWindowsRuntimeDir(path)
		if err != nil {
			return fmt.Errorf("open IPC directory %s: %w", path, err)
		}
		defer windows.CloseHandle(h)
		actual, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT,
			windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			return fmt.Errorf("read IPC directory permissions %s: %w", path, err)
		}
		if err := validateWindowsRuntimeSecurity(actual, user.User.Sid); err != nil {
			return fmt.Errorf("validate IPC directory %s: %w", path, err)
		}
	}
	return nil
}

// New sockets inherit the validated directory's restrictive ACL on Windows.
func secureUnixSocket(string) error {
	return nil
}

func validateSocketPath(path string) error {
	if len(path) > maxWindowsSocketPathBytes {
		return fmt.Errorf(
			"unix-socket path is %d bytes; Windows supports at most %d: %s",
			len(path), maxWindowsSocketPathBytes, path,
		)
	}
	return nil
}

func isUnixSocketAddrInUse(err error) bool {
	return errors.Is(err, windows.WSAEADDRINUSE)
}

func isUnixSocketUnsupported(err error) bool {
	// Older Windows versions can report WSAEINVAL when creating an AF_UNIX
	// socket. The same code from bind or connect is an operational failure,
	// not a reason for auto mode to disable IPC.
	var syscallErr *os.SyscallError
	if !errors.As(err, &syscallErr) || syscallErr.Syscall != "socket" {
		return false
	}
	return errors.Is(syscallErr.Err, windows.WSAEAFNOSUPPORT) || errors.Is(syscallErr.Err, windows.WSAEINVAL)
}

func isNoServer(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	return errors.Is(err, windows.WSAECONNREFUSED)
}
