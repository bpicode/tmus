package bar

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

func TestHorizontal(t *testing.T) {
	padLeftAndRight := lipgloss.Style{}.PaddingLeft(1).PaddingRight(1).PaddingChar(' ')
	tests := []struct {
		name   string
		hBar   HBar
		expect string
	}{
		{
			name:   "simple bar",
			hBar:   Horizontal(10),
			expect: "──────────",
		},
		{
			name:   "with zero width",
			hBar:   Horizontal(0),
			expect: "",
		},
		{
			name:   "with negative width",
			hBar:   Horizontal(-1),
			expect: "",
		},
		{
			name:   "bar with text, left placement",
			hBar:   Horizontal(15, WithText("Hello", PlacementLeft)),
			expect: "──Hello────────",
		},
		{
			name:   "bar with text, left placement, padding",
			hBar:   Horizontal(15, WithText(padLeftAndRight.Render("Hello"), PlacementLeft)),
			expect: "── Hello ──────",
		},
		{
			name:   "bar with text, right placement",
			hBar:   Horizontal(15, WithText("Hello", PlacementRight)),
			expect: "────────Hello──",
		},
		{
			name:   "bar with text, right placement, padding",
			hBar:   Horizontal(15, WithText(padLeftAndRight.Render("Hello"), PlacementRight)),
			expect: "────── Hello ──",
		},
		{
			name:   "text larger than width, center placement",
			hBar:   Horizontal(10, WithText("Hello, world!", PlacementCenter)),
			expect: "Hello, wo…",
		},
		{
			name:   "text larger than width, right placement",
			hBar:   Horizontal(10, WithText("Hello, world!", PlacementRight)),
			expect: "…o, world!",
		},
		{
			name:   "text larger than width, left placement",
			hBar:   Horizontal(10, WithText("Hello, world!", PlacementLeft)),
			expect: "Hello, wo…",
		},
		{
			name:   "no space for dashes, left placement",
			hBar:   Horizontal(10, WithText("helloworld", PlacementLeft)),
			expect: "helloworld",
		},
		{
			name:   "only space for one dash, left placement",
			hBar:   Horizontal(11, WithText("helloworld", PlacementLeft)),
			expect: "helloworld─",
		},
		{
			name:   "only space for two dashes, left placement",
			hBar:   Horizontal(12, WithText("helloworld", PlacementLeft)),
			expect: "─helloworld─",
		},
		{
			name:   "only space for three dashes, left placement",
			hBar:   Horizontal(13, WithText("helloworld", PlacementLeft)),
			expect: "─helloworld──",
		},
		{
			name:   "only space for four dashes, left placement",
			hBar:   Horizontal(14, WithText("helloworld", PlacementLeft)),
			expect: "──helloworld──",
		},
		{
			name:   "only space for five dashes, left placement",
			hBar:   Horizontal(15, WithText("helloworld", PlacementLeft)),
			expect: "──helloworld───",
		},
		{
			name:   "no space for dashes, right placement",
			hBar:   Horizontal(10, WithText("helloworld", PlacementRight)),
			expect: "helloworld",
		},
		{
			name:   "only space for one dash, right placement",
			hBar:   Horizontal(11, WithText("helloworld", PlacementRight)),
			expect: "─helloworld",
		},
		{
			name:   "only space for two dashes, right placement",
			hBar:   Horizontal(12, WithText("helloworld", PlacementRight)),
			expect: "─helloworld─",
		},
		{
			name:   "only space for three dashes, right placement",
			hBar:   Horizontal(13, WithText("helloworld", PlacementRight)),
			expect: "──helloworld─",
		},
		{
			name:   "only space for four dashes, right placement",
			hBar:   Horizontal(14, WithText("helloworld", PlacementRight)),
			expect: "──helloworld──",
		},
		{
			name:   "only space for five dashes, right placement",
			hBar:   Horizontal(15, WithText("helloworld", PlacementRight)),
			expect: "───helloworld──",
		},
		{
			name:   "no space for dashes, center placement",
			hBar:   Horizontal(10, WithText("helloworld", PlacementCenter)),
			expect: "helloworld",
		},
		{
			name:   "only space for one dash, center placement",
			hBar:   Horizontal(11, WithText("helloworld", PlacementCenter)),
			expect: "helloworld─",
		},
		{
			name:   "only space for two dashes, center placement",
			hBar:   Horizontal(12, WithText("helloworld", PlacementCenter)),
			expect: "─helloworld─",
		},
		{
			name:   "only space for three dashes, center placement",
			hBar:   Horizontal(13, WithText("helloworld", PlacementCenter)),
			expect: "─helloworld──",
		},
		{
			name:   "only space for four dashes, center placement",
			hBar:   Horizontal(14, WithText("helloworld", PlacementCenter)),
			expect: "──helloworld──",
		},
		{
			name:   "only space for five dashes, center placement",
			hBar:   Horizontal(15, WithText("helloworld", PlacementCenter)),
			expect: "──helloworld───",
		},
		{
			name:   "maxEdgePad zero, left placement",
			hBar:   Horizontal(15, WithText("Hello", PlacementLeft), WithMaxEdgePad(0)),
			expect: "Hello──────────",
		},
		{
			name:   "maxEdgePad zero, right placement",
			hBar:   Horizontal(15, WithText("Hello", PlacementRight), WithMaxEdgePad(0)),
			expect: "──────────Hello",
		},
		{
			name:   "maxEdgePad one, left placement",
			hBar:   Horizontal(15, WithText("Hello", PlacementLeft), WithMaxEdgePad(1)),
			expect: "─Hello─────────",
		},
		{
			name:   "maxEdgePad three, left placement, above crossover",
			hBar:   Horizontal(15, WithText("Hello", PlacementLeft), WithMaxEdgePad(3)),
			expect: "───Hello───────",
		},
		{
			name:   "maxEdgePad three, left placement, at crossover",
			hBar:   Horizontal(16, WithText("helloworld", PlacementLeft), WithMaxEdgePad(3)),
			expect: "───helloworld───",
		},
		{
			name:   "maxEdgePad three, left placement, below crossover",
			hBar:   Horizontal(15, WithText("helloworld", PlacementLeft), WithMaxEdgePad(3)),
			expect: "──helloworld───",
		},
		{
			name:   "maxEdgePad larger than available dashes",
			hBar:   Horizontal(15, WithText("helloworld", PlacementLeft), WithMaxEdgePad(50)),
			expect: "──helloworld───",
		},
		{
			name:   "negative maxEdgePad is clamped to zero",
			hBar:   Horizontal(15, WithText("Hello", PlacementLeft), WithMaxEdgePad(-5)),
			expect: "Hello──────────",
		},
		{
			name:   "center placement ignores maxEdgePad",
			hBar:   Horizontal(15, WithText("Hello", PlacementCenter), WithMaxEdgePad(0)),
			expect: "─────Hello─────",
		},
		{
			name:   "later options override earlier ones",
			hBar:   Horizontal(15, WithMaxEdgePad(3), WithMaxEdgePad(1), WithText("Hello", PlacementLeft)),
			expect: "─Hello─────────",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := tt.hBar.View()
			assert.Equal(t, tt.expect, view)
		})
	}
}

func TestHorizontalWithStyle(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	tests := []struct {
		name   string
		hBar   HBar
		expect string
	}{
		{
			name:   "plain bar",
			hBar:   Horizontal(10, WithStyle(styled)),
			expect: styled.Render(strings.Repeat("─", 10)),
		},
		{
			name:   "bar with text",
			hBar:   Horizontal(15, WithStyle(styled), WithText("Hello", PlacementLeft)),
			expect: styled.Render(strings.Repeat("─", 2)) + "Hello" + styled.Render(strings.Repeat("─", 8)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := tt.hBar.View()
			assert.Equal(t, tt.expect, view)
			assert.Equal(t, lipgloss.Width(tt.expect), lipgloss.Width(view))
		})
	}
}
