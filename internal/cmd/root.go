package cmd

import (
	"errors"
	"fmt"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/integration/ipc"
	"github.com/bpicode/tmus/internal/integration/mpris"
	"github.com/spf13/cobra"

	"github.com/bpicode/tmus/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "tmus [files...]",
	Short: "Terminal music player",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		startDir, _ := cmd.Flags().GetString("dir")
		cfg, err := loadConfigFromFlags(cmd)
		if err != nil {
			return err
		}

		ipcSession, err := ipc.Open(cfg.IPC, args)
		if err != nil {
			return err
		}
		if ipcSession.Handled() {
			return nil
		}
		defer collectErr(&err, "ipc", ipcSession.Close)

		playerApp := core.New(cfg)

		if err := ipcSession.Serve(playerApp); err != nil {
			return fmt.Errorf("serve ipc: %w", err)
		}

		if cfg.MPRIS.Enabled {
			if mprisSvc, _ := mpris.Start(playerApp); mprisSvc != nil {
				defer collectErr(&err, "mpris", mprisSvc.Close)
			}
		}

		if errUi := ui.Run(playerApp, startDir, cfg.TUI, args); errUi != nil {
			return fmt.Errorf("ui: %w", errUi)
		}
		return nil
	},
}

func collectErr(dst *error, label string, f func() error) {
	if err := f(); err != nil {
		*dst = errors.Join(*dst, fmt.Errorf("%s: %w", label, err))
	}
}

// Execute runs the root command.
func Execute() {
	err := rootCmd.Execute()
	cobra.CheckErr(err)
}

func init() {
	addConfigFlags(rootCmd)
	rootCmd.Flags().StringP("dir", "d", "", "starting directory for the file browser")
}
