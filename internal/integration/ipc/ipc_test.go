package ipc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/library"
	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSessionBackend struct {
	handle     requestHandler
	serveCount int
	closeCount int
	closeErr   error
}

type fakeRequestTarget struct {
	lib         *library.Library
	dispatchErr error
}

func (t fakeRequestTarget) Library() *library.Library {
	return t.lib
}

func (t fakeRequestTarget) Dispatch(core.Command) error {
	return t.dispatchErr
}

func (b *fakeSessionBackend) Serve(handle requestHandler) error {
	b.handle = handle
	b.serveCount++
	return nil
}

func (b *fakeSessionBackend) Close() error {
	b.closeCount++
	return b.closeErr
}

func TestOpenOffReturnsNoopSession(t *testing.T) {
	session, err := Open(config.IPCConfig{SingleInstance: config.SingleInstanceOff}, nil)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.False(t, session.Handled())
	assert.NoError(t, session.Serve(nil))
	assert.NoError(t, session.Close())
	assert.NoError(t, session.Close())
}

func TestOpenRejectsInvalidMode(t *testing.T) {
	session, err := Open(config.IPCConfig{SingleInstance: "magic"}, nil)
	assert.Nil(t, session)
	assert.ErrorContains(t, err, `unsupported single-instance mode "magic"`)
}

func TestSessionLifecycle(t *testing.T) {
	closeErr := errors.New("close failed")
	backend := &fakeSessionBackend{closeErr: closeErr}
	session := &Session{backend: backend}
	appRef := &core.App{}

	require.NoError(t, session.Serve(appRef))
	require.NotNil(t, backend.handle)
	assert.NoError(t, backend.handle(request{}))
	assert.Equal(t, 1, backend.serveCount)
	assert.ErrorIs(t, session.Serve(appRef), errAlreadyServed)

	assert.ErrorIs(t, session.Close(), closeErr)
	assert.ErrorIs(t, session.Close(), closeErr)
	assert.Equal(t, 1, backend.closeCount)
	assert.ErrorIs(t, session.Serve(appRef), errSessionClosed)
}

func TestHandledSessionCannotServe(t *testing.T) {
	session := &Session{handled: true}
	assert.True(t, session.Handled())
	assert.ErrorIs(t, session.Serve(&core.App{}), errAlreadyHanded)
}

func TestHandleRequestReportsDispatchError(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "song.mp3")
	require.NoError(t, os.WriteFile(audioPath, nil, 0o600))
	target := fakeRequestTarget{
		lib:         library.New(library.DefaultOptions()),
		dispatchErr: core.ErrAppClosed,
	}

	err := handleRequest(target, request{Paths: []string{audioPath}})
	assert.ErrorIs(t, err, core.ErrAppClosed)
	assert.ErrorContains(t, err, "dispatch handed-off tracks")
}

func TestBuildTracksPreservesStreamShortcutPath(t *testing.T) {
	dir := t.TempDir()
	streamPath := filepath.Join(dir, "radio.stream")
	err := os.WriteFile(streamPath, []byte("https://example.com/radio.mp3\n"), 0644)
	require.NoError(t, err)

	tracks := buildTracks(nil, []string{streamPath})

	assert.Equal(t, []core.Track{
		{Name: "radio.stream", Path: streamPath},
	}, tracks)
}

func TestNewRequestPreservesSenderPathMeaning(t *testing.T) {
	senderDir := t.TempDir()
	audioPath := filepath.Join(senderDir, "song.mp3")
	require.NoError(t, os.WriteFile(audioPath, nil, 0o600))
	t.Chdir(senderDir)

	req := newRequest([]string{"song.mp3", "https://example.com/radio.mp3"})
	t.Chdir(t.TempDir())

	assert.Equal(t, []string{
		audioPath,
		"https://example.com/radio.mp3",
	}, req.Paths)
	assert.Equal(t, []core.Track{
		{Name: "song.mp3", Path: audioPath},
	}, buildTracks(nil, req.Paths[:1]))
}

func TestBuildTracksNormalizesRelativePath(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "song.mp3")
	err := os.WriteFile(audioPath, nil, 0644)
	require.NoError(t, err)

	t.Chdir(dir)
	tracks := buildTracks(nil, []string{"song.mp3"})

	assert.Equal(t, []core.Track{
		{Name: "song.mp3", Path: audioPath},
	}, tracks)
}
