//go:build windows

package ipc

import "github.com/bpicode/tmus/internal/app/core"

// Server is a stub implementation on Windows.
type Server struct{}

// Send is not supported on Windows yet.
func Send(paths []string) error {
	_ = paths
	return ErrNotSupported
}

// StartServer is not supported on Windows yet.
func StartServer(appRef *core.App) (*Server, error) {
	_ = appRef
	return nil, ErrNotSupported
}

// Close shuts down the IPC listener.
func (s *Server) Close() error {
	return nil
}
