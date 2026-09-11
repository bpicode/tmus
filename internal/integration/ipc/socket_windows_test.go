//go:build windows

package ipc

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestWindowsUnixSocketUnsupportedErrors(t *testing.T) {
	tests := []struct {
		name string
		op   string
		err  error
		want bool
	}{
		{name: "unsupported family at socket creation", op: "socket", err: windows.WSAEAFNOSUPPORT, want: true},
		{name: "invalid argument at socket creation", op: "socket", err: windows.WSAEINVAL, want: true},
		{name: "invalid bind argument", op: "bind", err: windows.WSAEINVAL},
		{name: "invalid connect argument", op: "connect", err: windows.WSAEINVAL},
		{name: "unsupported family at bind", op: "bind", err: windows.WSAEAFNOSUPPORT},
		{name: "unsupported family at connect", op: "connect", err: windows.WSAEAFNOSUPPORT},
		{name: "socket access denied", op: "socket", err: windows.WSAEACCES},
		{name: "bind access denied", op: "bind", err: windows.WSAEACCES},
		{name: "connect access denied", op: "connect", err: windows.WSAEACCES},
		{name: "filesystem access denied", op: "connect", err: windows.ERROR_ACCESS_DENIED},
		{name: "connection refused", op: "connect", err: windows.WSAECONNREFUSED},
		{name: "address in use", op: "bind", err: windows.WSAEADDRINUSE},
		{name: "invalid pathname", op: "bind", err: syscall.EINVAL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := os.NewSyscallError(tt.op, tt.err)
			assert.Equal(t, tt.want, isUnixSocketUnsupported(err))
			// net wraps socket errors before they reach the IPC backend.
			wrapped := fmt.Errorf("open IPC: %w", &net.OpError{Op: "dial", Net: "unix", Err: err})
			assert.Equal(t, tt.want, isUnixSocketUnsupported(wrapped))
		})
	}
	// Bare codes do not identify which operation failed.
	for _, err := range []error{nil, windows.WSAEINVAL, windows.WSAEAFNOSUPPORT} {
		assert.False(t, isUnixSocketUnsupported(err))
	}
}

func TestWindowsIPCRuntimeDirUsesLocalAppData(t *testing.T) {
	localAppData := shortWindowsTempDir(t)
	t.Setenv("LocalAppData", localAppData)

	dir, err := ipcRuntimeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(localAppData, "tmus", "run"), dir)
}

func TestWindowsSocketPathLength(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "maximum ASCII length", path: strings.Repeat("a", maxWindowsSocketPathBytes)},
		{
			name: "ASCII path too long",
			path: strings.Repeat("a", maxWindowsSocketPathBytes+1),
			want: "is 108 bytes; Windows supports at most 107",
		},
		{
			name: "maximum multibyte length",
			path: strings.Repeat("é", maxWindowsSocketPathBytes/2) + "a",
		},
		{
			name: "multibyte path too long",
			path: strings.Repeat("é", maxWindowsSocketPathBytes/2+1),
			want: "is 108 bytes; Windows supports at most 107",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSocketPath(tt.path)
			if tt.want == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.want)
				assert.ErrorContains(t, err, tt.path)
			}
		})
	}
}

func TestWindowsSocketPathValidatesResolvedEndpoint(t *testing.T) {
	suffix := filepath.Join("tmus", "run", "tmus.sock")
	base := `C:\` + strings.Repeat("a", maxWindowsSocketPathBytes-len(suffix)-4)
	t.Setenv("LocalAppData", base)

	path, err := socketPath()
	require.NoError(t, err)
	assert.Len(t, path, maxWindowsSocketPathBytes)

	t.Setenv("LocalAppData", base+"a")
	_, err = socketPath()
	assert.ErrorContains(t, err, "is 108 bytes; Windows supports at most 107")
}

// Keep the socket path below sockaddr_un's small pathname limit. t.TempDir
// includes the test name and can exceed that limit on Windows.
func shortWindowsTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tmus-ipc-")
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll(dir)) })
	return dir
}

func setupSocketTest(t *testing.T) string {
	t.Helper()
	base := shortWindowsTempDir(t)
	t.Setenv("LocalAppData", base)
	return filepath.Join(base, "tmus", "run")
}
