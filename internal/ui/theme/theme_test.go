package theme

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestResolveConstructsTheme(t *testing.T) {
	th := Resolve(config.ThemeConfig{
		Preset:     "matrix",
		Primary:    "#abcdef",
		Background: terminalColor,
	})

	assert.Equal(t, lipgloss.Color("#abcdef"), th.Primary)
	assert.Equal(t, lipgloss.Color("#c1ff8a"), th.Secondary)
	assert.Nil(t, th.Background)
}
