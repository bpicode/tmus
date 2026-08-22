package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

func newVersionCmd(info *debug.BuildInfo) *cobra.Command {
	version := "unknown"
	if info != nil && info.Main.Version != "" {
		version = info.Main.Version
	}

	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "tmus %s\n", version); err != nil {
				return fmt.Errorf("write version: %w", err)
			}
			return nil
		},
	}
}

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		info = nil
	}
	rootCmd.AddCommand(newVersionCmd(info))
}
