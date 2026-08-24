package cmd

import (
	"os"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
)

var (
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Cyan).Bold(true)
	cyan       = lipgloss.NewStyle().Foreground(lipgloss.Cyan)
	white      = lipgloss.NewStyle().Foreground(lipgloss.BrightWhite).Bold(true)

	icon = "" +
		"              " + cyan.Render(".:::----:::.") + "              \n" +
		"          " + cyan.Render(".:-==============-:.") + "          \n" +
		"        " + cyan.Render(":-====================-:") + "        \n" +
		"      " + cyan.Render(":==========================:") + "      \n" +
		"     " + cyan.Render("-============-===============-") + "     \n" +
		"    " + cyan.Render("-==========-") + white.Render("+%*=") + cyan.Render("--=============-") + "    \n" +
		"   " + cyan.Render("-===========-") + white.Render("+@@@#+=") + cyan.Render("--===========-") + "   \n" +
		"  " + cyan.Render(".============-") + white.Render("+@@@@@@#+=") + cyan.Render("--=========.") + "  \n" +
		"  " + cyan.Render(":============-") + white.Render("+@@@@@@@@@#+") + cyan.Render("=========:") + "  \n" +
		"  " + cyan.Render(":============-") + white.Render("+@@@@@@@@@#+") + cyan.Render("=========:") + "  \n" +
		"  " + cyan.Render(".============-") + white.Render("+@@@@@@#+=") + cyan.Render("--=========.") + "  \n" +
		"   " + cyan.Render("-===========-") + white.Render("+@@@#+=") + cyan.Render("--===========-") + "   \n" +
		"    " + cyan.Render("-==========-") + white.Render("+%*=") + cyan.Render("--=============-") + "    \n" +
		"     " + cyan.Render("-============-===============-") + "     \n" +
		"      " + cyan.Render(":==========================:") + "      \n" +
		"        " + cyan.Render(":-====================-:") + "        \n" +
		"          " + cyan.Render(".:-==============-:.") + "          \n" +
		"              " + cyan.Render(".:::----:::.") + "              "

	aboutString = icon + "\n\n" +
		titleStyle.Render("tmus - terminal music player") + "\n\n" +
		"Homepage:       https://github.com/bpicode/tmus\n" +
		"Issue tracking: https://github.com/bpicode/tmus/issues\n" +
		"License:        GPL-3.0\n"
)

func newAboutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "about",
		Short: "About tmus",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
			_, err := w.Write([]byte(aboutString))
			return err
		},
	}
}

func init() {
	rootCmd.AddCommand(newAboutCmd())
}
