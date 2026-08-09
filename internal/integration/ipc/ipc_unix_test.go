//go:build !windows

package ipc

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestUnixSocketSessionHandsOffToPrimaryInstance(t *testing.T) {
	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	ipcCfg := config.IPCConfig{SingleInstance: config.SingleInstanceUnixSocket}

	primary, err := Open(ipcCfg, nil)
	require.NoError(t, err)
	require.False(t, primary.Handled())
	t.Cleanup(func() { assert.NoError(t, primary.Close()) })

	info, err := os.Stat(filepath.Join(runtimeDir, "tmus.sock"))
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
	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	session, err := Open(
		config.IPCConfig{SingleInstance: config.SingleInstanceAuto},
		nil,
	)
	require.NoError(t, err)
	require.False(t, session.Handled())
	t.Cleanup(func() { assert.NoError(t, session.Close()) })

	_, err = os.Stat(filepath.Join(runtimeDir, "tmus.sock"))
	assert.NoError(t, err)
}

func TestUnixSocketSessionReclaimsStaleEndpoint(t *testing.T) {
	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	socketPath := filepath.Join(runtimeDir, "tmus.sock")
	require.NoError(t, os.WriteFile(socketPath, nil, 0o600))

	session, err := Open(
		config.IPCConfig{SingleInstance: config.SingleInstanceUnixSocket},
		nil,
	)
	require.NoError(t, err)
	require.False(t, session.Handled())
	assert.NoError(t, session.Close())
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
