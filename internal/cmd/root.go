package cmd

import (
	"errors"
	"fmt"
	"os"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		startDir, _ := cmd.Flags().GetString("dir")
		cfg, err := loadConfigFromFlags(cmd)
		if err != nil {
			return err
		}

		if err := ipc.Send(args); err == nil {
			return nil
		} else if !errors.Is(err, ipc.ErrNoServer) && !errors.Is(err, ipc.ErrNotSupported) {
			return err
		}

		playerApp := core.New(cfg)
		ipcServer, err := ipc.StartServer(playerApp)
		if err != nil && !errors.Is(err, ipc.ErrAlreadyRunning) && !errors.Is(err, ipc.ErrNotSupported) {
			return err
		}
		if ipcServer != nil {
			defer ipcServer.Close()
		}
		if cfg.MPRIS.Enabled {
			if mprisSvc, _ := mpris.Start(playerApp); mprisSvc != nil {
				defer mprisSvc.Close()
			}
		}
		if err := ui.Run(playerApp, startDir, cfg, args); err != nil {
			return fmt.Errorf("ui: %w", err)
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "tmus:", err)
		os.Exit(1)
	}
}

func init() {
	addConfigFlags(rootCmd)
	rootCmd.Flags().StringP("dir", "d", "", "starting directory for the file browser")
}
