package tabs_test

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/tabs"
)

func ExampleNew() {
	keys := tabs.DefaultKeyMap()
	keys.Next.SetKeys("right")
	keys.Previous.SetKeys("left")
	styles := tabs.Styles{
		Border: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()),
	}
	m, err := tabs.New([]tabs.Tab{
		{ID: "one", Label: "One"},
		{ID: "two", Label: "Two"},
	}, tabs.WithKeyMap(keys), tabs.WithStyles(styles))
	if err != nil {
		fmt.Println(err)
		return
	}
	m.Focus()
	m.SetSize(20, 5)
	// A parent's Update would also return the command to Bubble Tea.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})

	// The parent owns the content and chooses what to render for the active tab.
	content := map[string]string{
		"one": "First tab content",
		"two": "Second tab content",
	}
	fmt.Println(m.Render(content[m.ActiveID()]))

	// Output:
	// ╭─────┬─────┬──────╮
	// │ One │ Two │      │
	// ├─────┴─────┴──────┤
	// │Second tab content│
	// ╰──────────────────╯
}
