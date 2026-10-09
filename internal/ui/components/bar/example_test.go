package bar_test

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/bar"
)

func ExampleHorizontal() {
	text := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1).Render("Hello")
	hBar := bar.Horizontal(15, bar.WithText(text, bar.PlacementLeft))
	fmt.Println(hBar.View())

	// Output:
	// ── Hello ──────
}
