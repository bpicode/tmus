//go:build !windows

package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/bpicode/tmus/internal/app/core"
)

type Server struct {
	ln   net.Listener
	path string
	mu   sync.Mutex
}

// Send tries to forward paths to an existing tmus instance.
// Returns ErrNoServer if no instance is running.
func Send(paths []string) error {
	socketPath, err := socketPath()
	if err != nil {
		return err
	}
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		if isNoServer(err) {
			return ErrNoServer
		}
		return err
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)
	if err := enc.Encode(request{Paths: paths}); err != nil {
		return err
	}
	var resp response
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

// StartServer listens for IPC requests to append tracks.
func StartServer(appRef *core.App) (*Server, error) {
	if appRef == nil {
		return nil, errors.New("app is nil")
	}
	socketPath, err := socketPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		return nil, err
	}

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			conn, dialErr := net.Dial("unix", socketPath)
			if dialErr == nil {
				_ = conn.Close()
				return nil, ErrAlreadyRunning
			}
			_ = os.Remove(socketPath)
			ln, err = net.Listen("unix", socketPath)
		}
	}
	if err != nil {
		return nil, err
	}

	server := &Server{ln: ln, path: socketPath}
	go server.serve(appRef)
	return server, nil
}

// Close shuts down the IPC listener.
func (s *Server) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln == nil {
		return nil
	}
	err := s.ln.Close()
	s.ln = nil
	if s.path != "" {
		_ = os.Remove(s.path)
	}
	return err
}

func (s *Server) serve(appRef *core.App) {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go handleConn(conn, appRef)
	}
}

func handleConn(conn net.Conn, appRef *core.App) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	var req request
	if err := dec.Decode(&req); err != nil {
		_ = enc.Encode(response{OK: false, Error: err.Error()})
		return
	}
	tracks := buildTracks(req.Paths)
	if len(tracks) > 0 {
		_ = appRef.Dispatch(core.Command{Type: core.CmdAddAll, Tracks: tracks})
	}
	_ = enc.Encode(response{OK: true})
}

func socketPath() (string, error) {
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "tmus.sock"), nil
	}
	base := os.TempDir()
	if base == "" {
		user, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		base = user
	}
	return filepath.Join(base, fmt.Sprintf("tmus-%d.sock", os.Getuid())), nil
}

func isNoServer(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such file") || strings.Contains(msg, "connection refused")
}
