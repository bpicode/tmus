package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	maxUnixSocketClaimAttempts = 3
	unixSocketRequestTimeout   = 5 * time.Second
	maxUnixSocketMessageBytes  = 2 * 1024 * 1024
)

type unixSocketSession struct {
	ln net.Listener

	mu       sync.Mutex
	closing  bool
	conns    map[net.Conn]struct{}
	workers  sync.WaitGroup
	serveErr error
}

type unixSocketResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func openUnixSocket(paths []string) (sessionBackend, bool, error) {
	socketPath, err := socketPath()
	if err != nil {
		return nil, false, err
	}

	if err := prepareRuntimeDir(filepath.Dir(socketPath)); err != nil {
		return nil, false, err
	}

	handled, err := tryUnixSocketHandoff(socketPath, paths)
	if err != nil {
		return nil, false, fmt.Errorf("handoff to %s: %w", socketPath, err)
	}
	if handled {
		return nil, true, nil
	}

	return claimUnixSocket(socketPath, paths)
}

// tryUnixSocketHandoff forwards paths when socketPath belongs to a reachable
// server. A missing or refused endpoint means no server and is not an error.
func tryUnixSocketHandoff(socketPath string, paths []string) (bool, error) {
	err := sendUnixSocket(socketPath, paths)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, errNoServer) {
		return false, nil
	}
	return false, err
}

// claimUnixSocket establishes which process is primary before Session.Serve
// starts accepting requests. If another process wins the bind race, the paths
// are handed to that process. Unreachable contended endpoints are treated as
// stale and removed only after their filesystem identity is rechecked.
func claimUnixSocket(socketPath string, paths []string) (sessionBackend, bool, error) {
	var lastErr error
	for range maxUnixSocketClaimAttempts {
		ln, err := listenUnixSocket(socketPath)
		if err == nil {
			return newUnixSocketSession(ln), false, nil
		}
		lastErr = err
		if !isUnixSocketAddrInUse(err) {
			return nil, false, fmt.Errorf("listen on %s: %w", socketPath, err)
		}

		contendedEndpoint, err := os.Lstat(socketPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, false, fmt.Errorf("inspect contended IPC endpoint: %w", err)
		}

		handled, err := tryUnixSocketHandoff(socketPath, paths)
		if err != nil {
			return nil, false, fmt.Errorf("handoff after endpoint contention: %w", err)
		}
		if handled {
			return nil, true, nil
		}

		if err := removeStaleUnixSocket(socketPath, contendedEndpoint); err != nil {
			return nil, false, err
		}
	}
	return nil, false, fmt.Errorf(
		"claim IPC endpoint after %d attempts: %w",
		maxUnixSocketClaimAttempts,
		lastErr,
	)
}

// removeStaleUnixSocket removes socketPath only when it still identifies the
// endpoint observed before the failed handoff. A missing or replaced endpoint
// is left alone so the claim loop can retry. The identity check narrows the
// unavoidable race between inspecting and unlinking a filesystem path.
func removeStaleUnixSocket(socketPath string, observed os.FileInfo) error {
	current, err := os.Lstat(socketPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reinspect contended IPC endpoint: %w", err)
	}
	if !os.SameFile(observed, current) {
		return nil
	}
	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale IPC endpoint: %w", err)
	}
	return nil
}

func sendUnixSocket(socketPath string, paths []string) error {
	conn, err := net.DialTimeout("unix", socketPath, unixSocketRequestTimeout)
	if err != nil {
		if isUnixSocketUnsupported(err) {
			return errors.Join(errNotSupported, err)
		}
		if isNoServer(err) {
			return errNoServer
		}
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(unixSocketRequestTimeout)); err != nil {
		return fmt.Errorf("set IPC request deadline: %w", err)
	}

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(io.LimitReader(conn, maxUnixSocketMessageBytes))
	if err := enc.Encode(newRequest(paths)); err != nil {
		return err
	}
	var resp unixSocketResponse
	if err := dec.Decode(&resp); err != nil {
		return err
	}
	if !resp.OK {
		if resp.Error != "" {
			return errors.New(resp.Error)
		}
		return errors.New("ipc request failed")
	}
	return nil
}

func listenUnixSocket(socketPath string) (*net.UnixListener, error) {
	addr := &net.UnixAddr{Name: socketPath, Net: "unix"}
	ln, err := net.ListenUnix("unix", addr)
	if err != nil {
		if isUnixSocketUnsupported(err) {
			return nil, errors.Join(errNotSupported, err)
		}
		return nil, err
	}
	ln.SetUnlinkOnClose(true)
	if err := secureUnixSocket(socketPath); err != nil {
		return nil, errors.Join(err, ln.Close())
	}
	return ln, nil
}

func newUnixSocketSession(ln net.Listener) *unixSocketSession {
	return &unixSocketSession{
		ln:    ln,
		conns: make(map[net.Conn]struct{}),
	}
}

func (s *unixSocketSession) Serve(handle requestHandler) error {
	s.workers.Go(func() {
		err := s.serve(handle)
		s.mu.Lock()
		s.serveErr = err
		s.mu.Unlock()
	})
	return nil
}

func (s *unixSocketSession) Close() error {
	s.mu.Lock()
	s.closing = true
	conns := make([]net.Conn, 0, len(s.conns))
	for conn := range s.conns {
		conns = append(conns, conn)
	}
	s.mu.Unlock()

	var closeErr error
	if err := s.ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		closeErr = fmt.Errorf("close IPC listener: %w", err)
	}
	for _, conn := range conns {
		// The handler may concurrently close the same connection. Either way,
		// Close unblocks its I/O and workers.Wait observes its completion.
		_ = conn.Close()
	}

	s.workers.Wait()
	s.mu.Lock()
	serveErr := s.serveErr
	s.mu.Unlock()
	return errors.Join(closeErr, serveErr)
}

func (s *unixSocketSession) serve(handle requestHandler) error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if s.isClosing() && errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept IPC connection: %w", err)
		}
		if !s.startHandler(conn, handle) {
			_ = conn.Close()
			return nil
		}
	}
}

func (s *unixSocketSession) startHandler(conn net.Conn, handle requestHandler) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return false
	}
	s.conns[conn] = struct{}{}
	s.workers.Go(func() {
		handleConn(conn, handle)
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
	})
	return true
}

func (s *unixSocketSession) isClosing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closing
}

func handleConn(conn net.Conn, handle requestHandler) {
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(unixSocketRequestTimeout)); err != nil {
		return
	}
	dec := json.NewDecoder(io.LimitReader(conn, maxUnixSocketMessageBytes))
	enc := json.NewEncoder(conn)
	var req request
	if err := dec.Decode(&req); err != nil {
		_ = enc.Encode(unixSocketResponse{Error: err.Error()})
		return
	}
	if err := handle(req); err != nil {
		_ = enc.Encode(unixSocketResponse{Error: err.Error()})
		return
	}
	_ = enc.Encode(unixSocketResponse{OK: true})
}

func socketPath() (string, error) {
	runtimeDir, err := ipcRuntimeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(runtimeDir, "tmus.sock")
	if err := validateSocketPath(path); err != nil {
		return "", err
	}
	return path, nil
}
