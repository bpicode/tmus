package cmd

import (
	"bytes"
	"errors"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCmd(t *testing.T) {
	tests := []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{
			name: "release",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v0.9.8"}},
			want: "tmus v0.9.8\n",
		},
		{
			name: "pseudo-version",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v0.9.9-0.20260822190949-3125f2b32184"}},
			want: "tmus v0.9.9-0.20260822190949-3125f2b32184\n",
		},
		{
			name: "dirty",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v0.9.9-0.20260822190949-3125f2b32184+dirty"}},
			want: "tmus v0.9.9-0.20260822190949-3125f2b32184+dirty\n",
		},
		{
			name: "nil information",
			want: "tmus unknown\n",
		},
		{
			name: "empty version",
			info: &debug.BuildInfo{},
			want: "tmus unknown\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			cmd := newVersionCmd(tt.info)
			cmd.SetOut(&out)

			require.NoError(t, cmd.Execute())
			assert.Equal(t, tt.want, out.String())
		})
	}
}

func TestVersionCmdRejectsArguments(t *testing.T) {
	cmd := newVersionCmd(&debug.BuildInfo{Main: debug.Module{Version: "v0.9.8"}})
	cmd.SetArgs([]string{"extra"})

	assert.Error(t, cmd.Execute())
}

func TestVersionCmdReturnsWriteError(t *testing.T) {
	cmd := newVersionCmd(&debug.BuildInfo{Main: debug.Module{Version: "v0.9.8"}})
	cmd.SetOut(errorWriter{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "write version")
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
