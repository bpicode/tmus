//go:build !windows

package ipc

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSocketTest(t *testing.T) string {
	t.Helper()
	base := privateTempDir(t)
	t.Setenv("XDG_RUNTIME_DIR", base)
	return filepath.Join(base, "tmus")
}

func TestUnixSocketPermissions(t *testing.T) {
	ipcDir := setupSocketTest(t)
	session, err := Open(config.IPCConfig{SingleInstance: config.SingleInstanceUnixSocket}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, session.Close()) })

	dirInfo, err := os.Stat(ipcDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	info, err := os.Stat(filepath.Join(ipcDir, "tmus.sock"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestUnixSocketSessionUsesPrivateTemporaryDirectoryWithoutXDG(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("TMPDIR", tempDir)

	session, err := Open(
		config.IPCConfig{SingleInstance: config.SingleInstanceUnixSocket},
		nil,
	)
	require.NoError(t, err)
	require.False(t, session.Handled())
	t.Cleanup(func() { assert.NoError(t, session.Close()) })

	ipcDir := filepath.Join(tempDir, fmt.Sprintf("tmus-%d", os.Getuid()))
	dirInfo, err := os.Stat(ipcDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	socketInfo, err := os.Stat(filepath.Join(ipcDir, "tmus.sock"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), socketInfo.Mode().Perm())
}

func TestUnixSocketSessionReclaimsStaleEndpoint(t *testing.T) {
	runtimeDir := privateTempDir(t)
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	ipcDir := filepath.Join(runtimeDir, "tmus")
	require.NoError(t, os.Mkdir(ipcDir, 0o700))
	socketPath := filepath.Join(ipcDir, "tmus.sock")
	require.NoError(t, os.WriteFile(socketPath, nil, 0o600))

	session, err := Open(
		config.IPCConfig{SingleInstance: config.SingleInstanceUnixSocket},
		nil,
	)
	require.NoError(t, err)
	require.False(t, session.Handled())
	assert.NoError(t, session.Close())
}

func TestOpenValidatesDirectoryBeforeHandoff(t *testing.T) {
	for _, mode := range []config.SingleInstanceMode{config.SingleInstanceAuto, config.SingleInstanceUnixSocket} {
		for _, redirected := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/redirected=%t", mode, redirected), func(t *testing.T) {
				base := privateTempDir(t)
				t.Setenv("XDG_RUNTIME_DIR", base)
				dir := filepath.Join(base, "tmus")
				actual := dir
				want := "mode 0700"
				if redirected {
					actual = privateTempDir(t)
					require.NoError(t, os.Symlink(actual, dir))
					want = "symbolic link"
				} else {
					require.NoError(t, os.Mkdir(dir, 0o700))
					require.NoError(t, os.Chmod(dir, 0o755))
				}
				ln, err := net.Listen("unix", filepath.Join(actual, "tmus.sock"))
				require.NoError(t, err)
				server := newUnixSocketSession(ln)
				t.Cleanup(func() { assert.NoError(t, server.Close()) })
				requests := make(chan request, 1)
				require.NoError(t, server.Serve(func(req request) error {
					requests <- req
					return nil
				}))
				session, err := Open(config.IPCConfig{SingleInstance: mode}, []string{"private.mp3"})
				if session != nil {
					defer session.Close()
				}
				assert.ErrorContains(t, err, want)
				assert.Nil(t, session)
				select {
				case <-requests:
					t.Fatal("request sent before validating the directory")
				default:
				}
			})
		}
	}
}

func TestIPCRuntimeDir(t *testing.T) {
	t.Run("uses application directory under XDG runtime", func(t *testing.T) {
		base := privateTempDir(t)
		t.Setenv("XDG_RUNTIME_DIR", base)

		dir, err := ipcRuntimeDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(base, "tmus"), dir)
	})

	t.Run("ignores relative XDG runtime", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_RUNTIME_DIR", "relative/runtime")
		t.Setenv("TMPDIR", base)

		dir, err := ipcRuntimeDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(base, fmt.Sprintf("tmus-%d", os.Getuid())), dir)
	})

	t.Run("rejects relative temporary directory", func(t *testing.T) {
		t.Setenv("XDG_RUNTIME_DIR", "")
		t.Setenv("TMPDIR", "relative/tmp")

		_, err := ipcRuntimeDir()
		assert.ErrorContains(t, err, "temporary directory is not absolute")
	})

	t.Run("rejects insecure XDG runtime", func(t *testing.T) {
		base := t.TempDir()
		require.NoError(t, os.Chmod(base, 0o755))
		t.Setenv("XDG_RUNTIME_DIR", base)

		_, err := ipcRuntimeDir()
		assert.ErrorContains(t, err, "runtime directory must have mode 0700")
	})
}

func privateTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o700))
	return dir
}

func TestPrepareRuntimeDir(t *testing.T) {
	t.Run("creates owner-only directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "tmus")
		require.NoError(t, prepareRuntimeDir(dir))

		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	})

	t.Run("rejects symbolic link", func(t *testing.T) {
		base := t.TempDir()
		target := filepath.Join(base, "target")
		link := filepath.Join(base, "tmus")
		require.NoError(t, os.Mkdir(target, 0o700))
		require.NoError(t, os.Symlink(target, link))

		assert.ErrorContains(t, prepareRuntimeDir(link), "must not be a symbolic link")
	})

	t.Run("rejects permissive directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "tmus")
		require.NoError(t, os.Mkdir(dir, 0o700))
		require.NoError(t, os.Chmod(dir, 0o755))

		assert.ErrorContains(t, prepareRuntimeDir(dir), "must have mode 0700")
	})
}

func TestRemoveStaleUnixSocket(t *testing.T) {
	t.Run("removes observed endpoint", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "tmus.sock")
		require.NoError(t, os.WriteFile(path, nil, 0o600))
		observed, err := os.Lstat(path)
		require.NoError(t, err)

		require.NoError(t, removeStaleUnixSocket(path, observed))
		_, err = os.Lstat(path)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("preserves replacement endpoint", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tmus.sock")
		replacement := filepath.Join(dir, "replacement.sock")
		require.NoError(t, os.WriteFile(path, []byte("observed"), 0o600))
		observed, err := os.Lstat(path)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(replacement, []byte("replacement"), 0o600))
		require.NoError(t, os.Rename(replacement, path))

		require.NoError(t, removeStaleUnixSocket(path, observed))
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "replacement", string(data))
	})
}
