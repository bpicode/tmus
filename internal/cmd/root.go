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

		handled, err := handoffIPC(args)
		if err != nil || handled {
			return err
		}

		playerApp := core.New(cfg)

		ipcServer, err := startIPCServer(playerApp)
		if err != nil {
			return err
		}
		if ipcServer != nil {
			defer collectErr(&err, "ipc", ipcServer.Close)
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

func handoffIPC(args []string) (bool, error) {
	err := ipc.Send(args)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ipc.ErrNoServer) || errors.Is(err, ipc.ErrNotSupported) {
		return false, nil
	}
	return false, err
}

func startIPCServer(appRef *core.App) (*ipc.Server, error) {
	server, err := ipc.StartServer(appRef)
	if err == nil {
		return server, nil
	}
	if errors.Is(err, ipc.ErrAlreadyRunning) || errors.Is(err, ipc.ErrNotSupported) {
		return nil, nil
	}
	return nil, err
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
