package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func newManCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:     "man DIRECTORY",
		Short:   "Generate man pages",
		Long:    "Generate a complete man-page tree in DIRECTORY.",
		Example: "  tmus man /tmp/tmus-man && man -l /tmp/tmus-man/tmus.1",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := args[0]
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create man page directory: %w", err)
			}

			root.DisableAutoGenTag = true
			header := &doc.GenManHeader{
				Section: "1",
				Source:  "tmus",
				Manual:  "tmus Manual",
			}
			if err := doc.GenManTree(root, header, dir); err != nil {
				return fmt.Errorf("generate man pages: %w", err)
			}
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(newManCmd(rootCmd))
}
