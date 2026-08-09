//go:build !windows

package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	maxUnixSocketClaimAttempts = 3
	unixSocketRequestTimeout   = 5 * time.Second
)

type unixSocketSession struct {
	ln *net.UnixListener
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

	handled, err := tryUnixSocketHandoff(socketPath, paths)
	if err != nil {
		return nil, false, fmt.Errorf("handoff to %s: %w", socketPath, err)
	}
	if handled {
		return nil, true, nil
	}

	if err := prepareRuntimeDir(filepath.Dir(socketPath)); err != nil {
		return nil, false, err
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
			return &unixSocketSession{ln: ln}, false, nil
		}
		lastErr = err
		if !errors.Is(err, syscall.EADDRINUSE) {
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
	dec := json.NewDecoder(conn)
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
		return nil, err
	}
	ln.SetUnlinkOnClose(true)
	if err := os.Chmod(socketPath, 0o600); err != nil {
		return nil, errors.Join(err, ln.Close())
	}
	return ln, nil
}

func (s *unixSocketSession) Serve(handle requestHandler) error {
	go serveUnixSocket(s.ln, handle)
	return nil
}

func (s *unixSocketSession) Close() error {
	return s.ln.Close()
}

func serveUnixSocket(ln net.Listener, handle requestHandler) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleConn(conn, handle)
	}
}

func handleConn(conn net.Conn, handle requestHandler) {
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(unixSocketRequestTimeout)); err != nil {
		return
	}
	dec := json.NewDecoder(conn)
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
	return filepath.Join(runtimeDir, "tmus.sock"), nil
}

func ipcRuntimeDir() (string, error) {
	if base := os.Getenv("XDG_RUNTIME_DIR"); filepath.IsAbs(base) {
		if err := validatePrivateRuntimeDir(base); err != nil {
			return "", fmt.Errorf("validate XDG_RUNTIME_DIR: %w", err)
		}
		return filepath.Join(base, "tmus"), nil
	}

	base := os.TempDir()
	if !filepath.IsAbs(base) {
		return "", fmt.Errorf("temporary directory is not absolute: %s", base)
	}
	return filepath.Join(base, fmt.Sprintf("tmus-%d", os.Getuid())), nil
}

// prepareRuntimeDir creates and validates the directory that contains tmus IPC
// resources. Refusing symlinks, foreign ownership, and group or other access
// keeps the socket private from the moment it is created.
func prepareRuntimeDir(dir string) error {
	created := false
	if err := os.Mkdir(dir, 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create IPC directory: %w", err)
		}
	} else {
		created = true
	}
	if created {
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("secure IPC directory: %w", err)
		}
	}
	if err := validatePrivateRuntimeDir(dir); err != nil {
		return fmt.Errorf("validate IPC directory: %w", err)
	}
	return nil
}

func validatePrivateRuntimeDir(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect runtime directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("runtime directory must not be a symbolic link: %s", dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("runtime path is not a directory: %s", dir)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("inspect runtime directory ownership: %s", dir)
	}
	if int(stat.Uid) != os.Getuid() {
		return fmt.Errorf("runtime directory is not owned by the current user: %s", dir)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		return fmt.Errorf("runtime directory must have mode 0700, got %04o: %s", perm, dir)
	}
	return nil
}

func isNoServer(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	return errors.Is(err, syscall.ECONNREFUSED)
}
