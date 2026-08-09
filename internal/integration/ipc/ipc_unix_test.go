//go:build !windows

package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubListener struct {
	accept func() (net.Conn, error)
	close  func() error
}

func (l stubListener) Accept() (net.Conn, error) {
	return l.accept()
}

func (l stubListener) Close() error {
	return l.close()
}

func (stubListener) Addr() net.Addr {
	return &net.UnixAddr{Name: "test", Net: "unix"}
}

func TestUnixSocketEncodesHandlerError(t *testing.T) {
	server, client := net.Pipe()
	t.Cleanup(func() { _ = server.Close() })
	t.Cleanup(func() { _ = client.Close() })
	handlerErr := errors.New("handler failed")
	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, func(request) error { return handlerErr })
	}()

	require.NoError(t, json.NewEncoder(client).Encode(request{}))
	var resp unixSocketResponse
	require.NoError(t, json.NewDecoder(client).Decode(&resp))
	<-done

	assert.False(t, resp.OK)
	assert.Equal(t, handlerErr.Error(), resp.Error)
}

func TestUnixSocketSessionCloseWaitsForActiveHandler(t *testing.T) {
	server, client := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })

	listenerClosed := make(chan struct{})
	var closeListener sync.Once
	accepted := false
	ln := stubListener{
		accept: func() (net.Conn, error) {
			if !accepted {
				accepted = true
				return server, nil
			}
			<-listenerClosed
			return nil, net.ErrClosed
		},
		close: func() error {
			closeListener.Do(func() { close(listenerClosed) })
			return nil
		},
	}

	handling := make(chan struct{})
	release := make(chan struct{})
	released := false
	t.Cleanup(func() {
		if !released {
			close(release)
		}
	})
	session := newUnixSocketSession(ln)
	require.NoError(t, session.Serve(func(request) error {
		close(handling)
		<-release
		return nil
	}))

	require.NoError(t, json.NewEncoder(client).Encode(request{}))
	<-handling
	closed := make(chan error, 1)
	go func() { closed <- session.Close() }()

	require.NoError(t, client.SetReadDeadline(time.Now().Add(time.Second)))
	var resp unixSocketResponse
	assert.Error(t, json.NewDecoder(client).Decode(&resp))
	select {
	case err := <-closed:
		t.Fatalf("Close returned while the handler was active: %v", err)
	default:
	}

	close(release)
	released = true
	assert.NoError(t, <-closed)
}

func TestUnixSocketSessionCloseReportsAcceptError(t *testing.T) {
	acceptErr := errors.New("accept failed")
	session := newUnixSocketSession(stubListener{
		accept: func() (net.Conn, error) { return nil, acceptErr },
		close:  func() error { return nil },
	})
	require.NoError(t, session.Serve(func(request) error { return nil }))

	err := session.Close()
	assert.ErrorIs(t, err, acceptErr)
	assert.ErrorContains(t, err, "accept IPC connection")
}

func TestUnixSocketSessionHandsOffToPrimaryInstance(t *testing.T) {
	runtimeDir := privateTempDir(t)
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	ipcCfg := config.IPCConfig{SingleInstance: config.SingleInstanceUnixSocket}

	primary, err := Open(ipcCfg, nil)
	require.NoError(t, err)
	require.False(t, primary.Handled())
	t.Cleanup(func() { assert.NoError(t, primary.Close()) })

	ipcDir := filepath.Join(runtimeDir, "tmus")
	dirInfo, err := os.Stat(ipcDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	info, err := os.Stat(filepath.Join(ipcDir, "tmus.sock"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	appCfg := config.Default()
	appCfg.Cache.Dir = t.TempDir()
	appCfg.Lyrics.LrcLib.Enabled = false
	appRef := core.New(appCfg)
	t.Cleanup(appRef.ShutdownAndWait)
	require.NoError(t, primary.Serve(appRef))

	audioPath := filepath.Join(t.TempDir(), "song.mp3")
	require.NoError(t, os.WriteFile(audioPath, nil, 0o600))
	secondary, err := Open(ipcCfg, []string{audioPath})
	require.NoError(t, err)
	require.True(t, secondary.Handled())
	require.NoError(t, secondary.Close())

	assert.Eventually(t, func() bool {
		playlist := appRef.State().Playlist
		return len(playlist) == 1 && playlist[0].Path == audioPath
	}, time.Second, 10*time.Millisecond)
}

func TestAutoClaimsUnixSocketWhenSupported(t *testing.T) {
	runtimeDir := privateTempDir(t)
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	session, err := Open(
		config.IPCConfig{SingleInstance: config.SingleInstanceAuto},
		nil,
	)
	require.NoError(t, err)
	require.False(t, session.Handled())
	t.Cleanup(func() { assert.NoError(t, session.Close()) })

	_, err = os.Stat(filepath.Join(runtimeDir, "tmus", "tmus.sock"))
	assert.NoError(t, err)
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
