package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRootCmdAndAllSubCmdsHaveUse(t *testing.T) {
	assertHasUseRecursively(t, rootCmd)
}

func assertHasUseRecursively(t *testing.T, cmd *cobra.Command) {
	t.Run(cmd.Name(), func(t *testing.T) {
		assert.NotEmpty(t, cmd.Use, "Command '%s' has no 'use'", cmd.Name())
		for _, subCmd := range cmd.Commands() {
			assertHasUseRecursively(t, subCmd)
		}
	})
}

func TestRootCmdAndAllSubCmdsHaveShort(t *testing.T) {
	assertHasShortRecursively(t, rootCmd)
}

func assertHasShortRecursively(t *testing.T, cmd *cobra.Command) {
	t.Run(cmd.Name(), func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short, "Command '%s' has no 'short'", cmd.Name())
		for _, subCmd := range cmd.Commands() {
			assertHasShortRecursively(t, subCmd)
		}
	})
}
