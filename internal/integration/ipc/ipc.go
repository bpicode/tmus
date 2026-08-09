// Package ipc coordinates single-instance handoff between tmus processes.
package ipc

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/library"
	"github.com/bpicode/tmus/internal/config"
)

var (
	errNoServer      = errors.New("no running tmus instance")
	errNotSupported  = errors.New("ipc method not supported")
	errSessionClosed = errors.New("ipc session is closed")
	errAlreadyServed = errors.New("ipc session is already serving")
	errAlreadyHanded = errors.New("ipc request was already handed off")
)

type sessionBackend interface {
	Serve(requestHandler) error
	// Close stops accepting requests and waits for active handlers to return.
	Close() error
}

// requestHandler owns the application semantics of an IPC request. Backends
// are responsible for delivering requests and representing returned errors.
type requestHandler func(request) error

// requestTarget is the application surface needed to apply a handoff request.
type requestTarget interface {
	Library() *library.Library
	Dispatch(core.Command) error
}

// Session represents the result of negotiating single-instance IPC.
//
// Open always returns a non-nil Session when it succeeds. If Handled reports
// true, the arguments were forwarded to another process and the caller should
// exit. Otherwise, the caller should defer Close and call Serve after creating
// the application. Serve is a no-op when IPC is off or auto finds no supported
// method. A Session must not be copied after first use.
type Session struct {
	backend sessionBackend
	handled bool

	mu       sync.Mutex
	served   bool
	closed   bool
	closeErr error
}

// Open forwards paths to a running instance or claims the configured IPC
// endpoint for the caller. Endpoint ownership is established before Open
// returns, preventing another process from becoming the primary instance in
// the interval before Serve is called.
func Open(cfg config.IPCConfig, paths []string) (*Session, error) {
	switch cfg.SingleInstance {
	case config.SingleInstanceOff:
		return &Session{}, nil
	case config.SingleInstanceAuto:
		backend, handled, err := openUnixSocket(paths)
		if errors.Is(err, errNotSupported) {
			return &Session{}, nil
		}
		if err != nil {
			return nil, fmt.Errorf("open single-instance IPC automatically: %w", err)
		}
		return &Session{backend: backend, handled: handled}, nil
	case config.SingleInstanceUnixSocket:
		backend, handled, err := openUnixSocket(paths)
		if err != nil {
			return nil, fmt.Errorf("open single-instance IPC using unix-socket: %w", err)
		}
		return &Session{backend: backend, handled: handled}, nil
	default:
		return nil, fmt.Errorf("unsupported single-instance mode %q", cfg.SingleInstance)
	}
}

// Handled reports whether Open forwarded the request to a running process.
func (s *Session) Handled() bool {
	return s != nil && s.handled
}

// Serve starts handling handoff requests for the application. It may be called
// at most once and must not be called when Handled reports true.
func (s *Session) Serve(appRef *core.App) error {
	if s == nil {
		return errors.New("ipc session is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errSessionClosed
	}
	if s.handled {
		return errAlreadyHanded
	}
	if s.served {
		return errAlreadyServed
	}
	if s.backend != nil && appRef == nil {
		return errors.New("app is nil")
	}

	s.served = true
	if s.backend == nil {
		return nil
	}
	return s.backend.Serve(func(req request) error {
		return handleRequest(appRef, req)
	})
}

// Close stops IPC handling and releases any endpoint claimed by the session.
// When it returns, no handoff request can still call into the application.
// Close is idempotent; a nil Session is also safe to close.
func (s *Session) Close() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return s.closeErr
	}
	s.closed = true
	if s.backend != nil {
		s.closeErr = s.backend.Close()
	}
	return s.closeErr
}

type request struct {
	Paths []string `json:"paths"`
}

// newRequest resolves filesystem paths in the sending process so relative
// arguments keep their meaning after crossing into a process with another
// working directory. URLs remain unchanged.
func newRequest(paths []string) request {
	normalized := make([]string, len(paths))
	for i, path := range paths {
		normalized[i] = normalizeInputPath(path)
	}
	return request{Paths: normalized}
}

// handleRequest applies transport-independent handoff behavior. Keeping it in
// the common package ensures every backend resolves and dispatches paths in the
// same way.
func handleRequest(appRef requestTarget, req request) error {
	tracks := buildTracks(appRef.Library(), req.Paths)
	if len(tracks) == 0 {
		return nil
	}
	if err := appRef.Dispatch(core.Command{Type: core.CmdAddAll, Tracks: tracks}); err != nil {
		return fmt.Errorf("dispatch handed-off tracks: %w", err)
	}
	return nil
}

func buildTracks(lib *library.Library, paths []string) []core.Track {
	if len(paths) == 0 {
		return nil
	}
	if lib == nil {
		lib = library.New(library.DefaultOptions())
	}
	tracks := make([]core.Track, 0, len(paths))
	for _, value := range paths {
		entry, err := lib.EntryFromPath(normalizeInputPath(value))
		if err == nil && entry.IsAudio() {
			tracks = append(tracks, core.Track{
				Name: entry.Name(),
				Path: entry.Path(),
			})
		}
	}
	return tracks
}

func normalizeInputPath(value string) string {
	if value == "" || strings.Contains(value, "://") {
		return value
	}
	path := filepath.Clean(value)
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}
