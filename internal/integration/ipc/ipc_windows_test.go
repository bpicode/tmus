//go:build windows

package ipc

import (
	"testing"

	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoFallsBackWhenUnixSocketsAreUnsupported(t *testing.T) {
	session, err := Open(
		config.IPCConfig{SingleInstance: config.SingleInstanceAuto},
		nil,
	)
	require.NoError(t, err)
	assert.False(t, session.Handled())
	assert.NoError(t, session.Serve(nil))
	assert.NoError(t, session.Close())
}

func TestExplicitUnixSocketFailsWhenUnsupported(t *testing.T) {
	session, err := Open(
		config.IPCConfig{SingleInstance: config.SingleInstanceUnixSocket},
		nil,
	)
	assert.Nil(t, session)
	assert.ErrorIs(t, err, errNotSupported)
}
