package ipc

import (
	"encoding/json"
	"errors"
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
	setupSocketTest(t)
	ipcCfg := config.IPCConfig{SingleInstance: config.SingleInstanceUnixSocket}

	primary, err := Open(ipcCfg, nil)
	if errors.Is(err, errNotSupported) {
		t.Skip("platform does not support Unix-domain sockets")
	}
	require.NoError(t, err)
	require.False(t, primary.Handled())
	t.Cleanup(func() { assert.NoError(t, primary.Close()) })

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
	ipcDir := setupSocketTest(t)

	session, err := Open(
		config.IPCConfig{SingleInstance: config.SingleInstanceAuto},
		nil,
	)
	require.NoError(t, err)
	require.False(t, session.Handled())
	t.Cleanup(func() { assert.NoError(t, session.Close()) })
	if session.backend == nil {
		t.Skip("platform does not support Unix-domain sockets")
	}

	_, err = os.Stat(filepath.Join(ipcDir, "tmus.sock"))
	assert.NoError(t, err)
}
