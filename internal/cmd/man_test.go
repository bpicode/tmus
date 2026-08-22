package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManCmd(t *testing.T) {
	root := &cobra.Command{Use: "test", Short: "Test command"}
	root.AddCommand(newManCmd(root))
	dir := t.TempDir()
	root.SetArgs([]string{"man", dir})

	require.NoError(t, root.Execute())
	data, err := os.ReadFile(filepath.Join(dir, "test.1"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "test - Test command")
	data, err = os.ReadFile(filepath.Join(dir, "test-man.1"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `\fBtest man DIRECTORY [flags]\fP`)
}

func TestManCmdRejectsInvalidArguments(t *testing.T) {
	cmd := newManCmd(&cobra.Command{Use: "test", Short: "Test command"})

	assert.Error(t, cmd.Execute())
}

func TestManCmdReturnsDirectoryError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, nil, 0o644))
	cmd := newManCmd(&cobra.Command{Use: "test", Short: "Test command"})

	err := cmd.RunE(cmd, []string{filepath.Join(path, "manpages")})
	require.Error(t, err)
	assert.ErrorContains(t, err, "create man page directory")
}
