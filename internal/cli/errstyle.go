// Colorized command-error output. Cobra's own error printer is silenced
// (rootCmd.SilenceErrors, set in root.go) so this is the single place a
// command error reaches the terminal, styled when it's worth styling.
package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// flagNoColor backs the --no-color persistent flag (registered in root.go).
var flagNoColor bool

var errorPrefixStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1")) // red

// shouldColorizeError reports whether a command error should get a
// lipgloss-styled prefix instead of the old plain-text one. Every input is
// checked independently, same as the picker/spinner gates: --no-color and
// NO_COLOR (https://no-color.org) each independently turn it off, and stderr
// must be a real terminal regardless of either -- a script capturing stderr
// must never see ANSI codes even if it forgot both.
func shouldColorizeError(noColorFlag bool, noColorEnv string, stderrIsTTY bool) bool {
	if noColorFlag || noColorEnv != "" {
		return false
	}
	return stderrIsTTY
}

// PrintError writes a command's terminal error to stderr: "snowstorm:
// <message>" (byte-for-byte what main.go always printed) unless
// shouldColorizeError says this is a real interactive terminal, in which
// case it's a bold-red "Error:" prefix instead -- the one visible piece of
// this task's "humans get the nice UX, agents get the unchanged plain path"
// philosophy that lives outside any single command.
func PrintError(err error) {
	if err == nil {
		return
	}
	stderrIsTTY := term.IsTerminal(int(os.Stderr.Fd()))
	if shouldColorizeError(flagNoColor, os.Getenv("NO_COLOR"), stderrIsTTY) {
		fmt.Fprintln(os.Stderr, errorPrefixStyle.Render("Error:"), err)
		return
	}
	fmt.Fprintln(os.Stderr, "snowstorm:", err)
}
