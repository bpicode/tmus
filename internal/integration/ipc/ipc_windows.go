//go:build windows

package ipc

func openUnixSocket(paths []string) (sessionBackend, bool, error) {
	_ = paths
	return nil, false, errNotSupported
}
