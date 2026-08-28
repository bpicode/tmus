//go:build windows

package ipc

func openUnixSocket(_ []string) (sessionBackend, bool, error) {
	return nil, false, errNotSupported
}
