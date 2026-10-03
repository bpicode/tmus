package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/bpicode/tmus/internal/ui/view"
)

// Run2 starts the experimental tabbed layout preview.
func Run2(cfg config.TUIConfig) error {
	m, err := view.NewModel2(theme.Resolve(cfg.Theme))
	if err != nil {
		return fmt.Errorf("create experimental view: %w", err)
	}
	_, err = tea.NewProgram(m, tea.WithFPS(cfg.FPS)).Run()
	return err
}
