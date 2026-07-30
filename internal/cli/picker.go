// Interactive saved-query picker: an fzf-style fuzzy list shown by `snowstorm
// query` (bare, at a real terminal) instead of falling through to stdin, so
// a human running it with zero flags can browse and pick a saved query.
//
// This is entirely additive to the existing resolveSQL stdin path (query.go)
// -- when the picker doesn't apply (any of SQL arg/--file/--saved given,
// --skip-pick set, or either stdin/stdout isn't a real terminal), behavior
// falls through to resolveSQL completely unchanged, which is the
// agent/script path this task must never disturb.
package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// shouldShowPicker reports whether `query` (invoked bare) should show the
// interactive saved-query picker instead of reading stdin. It's pure and
// TTY/flag state is passed in explicitly so the decision matrix -- the part
// that actually matters for the "never block a non-interactive caller" rule
// -- is unit-testable without a real terminal or bubbletea program.
//
// The picker only applies when NONE of [SQL arg, --file, --saved] was given
// (those already fully determine the SQL text, in that priority order) AND
// --skip-pick wasn't passed AND both stdin and stdout are real terminals.
// Either TTY check failing means a script/CI/pipe caller, which must fall
// through to the existing stdin-reading behavior even if it forgot
// --skip-pick -- the flag is a courtesy on top of this hard gate, not the
// only guard.
func shouldShowPicker(hasArg, hasFile, hasSaved, skipPick, stdinIsTTY, stdoutIsTTY bool) bool {
	if hasArg || hasFile || hasSaved || skipPick {
		return false
	}
	return stdinIsTTY && stdoutIsTTY
}

// resolveSQLOrPick is resolveSQL's entry point plus the picker: it decides
// whether to show the picker at all (shouldShowPicker), and if not, defers
// to resolveSQL exactly as before -- that function and its stdin-reading
// last resort are untouched by this file.
//
// cancelled reports a clean user-initiated cancel (Ctrl-C/Esc in the
// picker, or a picker shown over an empty query dir): the caller should
// exit 0 with no output and no error, not print anything.
func resolveSQLOrPick(args []string) (sqlText string, cancelled bool, err error) {
	hasArg := len(args) == 1
	hasFile := flagQueryFile != ""
	hasSaved := flagQuerySaved != ""

	stdinIsTTY := term.IsTerminal(int(os.Stdin.Fd()))
	stdoutIsTTY := term.IsTerminal(int(os.Stdout.Fd()))

	if !shouldShowPicker(hasArg, hasFile, hasSaved, flagQuerySkipPick, stdinIsTTY, stdoutIsTTY) {
		sqlText, err = resolveSQL(args)
		return sqlText, false, err
	}

	items, err := listSavedQueries()
	if err != nil {
		return "", false, err
	}
	if len(items) == 0 {
		// Nothing to pick from -- same "no output, exit 0" contract as an
		// explicit cancel rather than an error, since this isn't a mistake
		// on the caller's part.
		return "", true, nil
	}

	name, cancelled, err := pickSavedQuery(items)
	if err != nil || cancelled {
		return "", cancelled, err
	}

	sqlText, err = resolveSavedQuery(name)
	return sqlText, false, err
}

// queryItem adapts a savedQueryInfo into bubbles/list's Item+DefaultItem
// interfaces: FilterValue feeds sahilm/fuzzy's subsequence matching (name
// and description both searchable), Title/Description drive the two-line
// rendering of list.NewDefaultDelegate().
type queryItem struct {
	name        string
	description string
}

func (i queryItem) FilterValue() string { return i.name + " " + i.description }
func (i queryItem) Title() string       { return i.name }
func (i queryItem) Description() string { return i.description }

// pickerModel is the bubbletea model backing the saved-query picker: mostly
// a thin wrapper around bubbles/list.Model, with its own Enter/Esc/Ctrl-C
// handling layered on top (list.Model has no notion of "the user is done
// picking" on its own).
type pickerModel struct {
	list      list.Model
	chosen    string
	cancelled bool
}

func newPickerModel(items []savedQueryInfo) pickerModel {
	listItems := make([]list.Item, len(items))
	for i, it := range items {
		listItems[i] = queryItem{name: it.Name, description: it.Description}
	}

	const width, height = 80, 20
	l := list.New(listItems, list.NewDefaultDelegate(), width, height)
	l.Title = "Saved queries (type to fuzzy-filter, enter to run, esc/ctrl+c to cancel)"
	l.SetShowStatusBar(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)

	return pickerModel{list: l}
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		case tea.KeyEnter:
			// Enter always runs the currently-highlighted item, whether or
			// not the user has stopped typing a filter -- fzf-style, not
			// "press enter once to accept the filter, again to select".
			if it, ok := m.list.SelectedItem().(queryItem); ok {
				m.chosen = it.name
			} else {
				m.cancelled = true
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m pickerModel) View() string {
	return "\n" + m.list.View()
}

// pickSavedQuery runs the interactive picker over items and returns the
// chosen entry's Name. Only called once shouldShowPicker has already
// confirmed both stdin and stdout are real terminals, so bubbletea's default
// os.Stdin/os.Stdout program I/O is safe to use as-is.
func pickSavedQuery(items []savedQueryInfo) (name string, cancelled bool, err error) {
	p := tea.NewProgram(newPickerModel(items))
	finalModel, err := p.Run()
	if err != nil {
		return "", false, fmt.Errorf("saved-query picker: %w", err)
	}

	fm, ok := finalModel.(pickerModel)
	if !ok || fm.cancelled || fm.chosen == "" {
		return "", true, nil
	}
	return fm.chosen, false, nil
}
