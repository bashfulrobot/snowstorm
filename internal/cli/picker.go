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
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
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
// interfaces: FilterValue is the text matchPrefix compares against (name
// first, then description), Title/Description drive the two-line rendering
// of list.NewDefaultDelegate().
type queryItem struct {
	name        string
	description string
}

func (i queryItem) FilterValue() string { return i.name + " " + i.description }
func (i queryItem) Title() string       { return i.name }
func (i queryItem) Description() string { return i.description }

// matchPrefix returns the items whose FilterValue starts with term
// (case-insensitive, whole pattern). An empty term matches nothing: the picker
// opens blank so a screen share never lists every customer's query name.
func matchPrefix(items []list.Item, term string) []list.Item {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return nil
	}
	var out []list.Item
	for _, it := range items {
		if strings.HasPrefix(strings.ToLower(it.FilterValue()), term) {
			out = append(out, it)
		}
	}
	return out
}

// pickerModel is the bubbletea model backing the saved-query picker: a text
// input plus a bubbles/list.Model that shows only the prefix matches for the
// typed text. The list's built-in fuzzy filter is off because it lists every
// item while the filter is empty.
type pickerModel struct {
	input     textinput.Model
	list      list.Model
	all       []list.Item
	chosen    string
	cancelled bool
}

func newPickerModel(items []savedQueryInfo) pickerModel {
	all := make([]list.Item, len(items))
	for i, it := range items {
		all[i] = queryItem{name: it.Name, description: it.Description}
	}

	const width, height = 80, 20
	l := list.New(nil, list.NewDefaultDelegate(), width, height-3)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	in := textinput.New()
	in.Prompt = "> "
	in.Focus()

	return pickerModel{input: in, list: l, all: all}
}

func (m pickerModel) Init() tea.Cmd { return textinput.Blink }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-3)
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		case tea.KeyEnter:
			// Enter runs the highlighted match. With nothing typed there is
			// nothing highlighted, so Enter is ignored rather than cancelling.
			if it, ok := m.list.SelectedItem().(queryItem); ok {
				m.chosen = it.name
				return m, tea.Quit
			}
			return m, nil
		case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		case tea.KeyCtrlP:
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(tea.KeyMsg{Type: tea.KeyUp})
			return m, cmd
		case tea.KeyCtrlN:
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(tea.KeyMsg{Type: tea.KeyDown})
			return m, cmd
		}
	}

	prev := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != prev {
		m.list.SetItems(matchPrefix(m.all, m.input.Value()))
		m.list.ResetSelected()
	}
	return m, cmd
}

func (m pickerModel) View() string {
	header := lipgloss.NewStyle().Bold(true).Render("Saved queries (type the start of a name, enter to run, esc/ctrl+c to cancel)")
	body := "Type to search. Nothing is listed until you do."
	if strings.TrimSpace(m.input.Value()) != "" {
		body = m.list.View()
	}
	return "\n" + header + "\n" + m.input.View() + "\n\n" + body
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
