// Progress spinner shown on stderr while a --format table query runs. Purely
// cosmetic and stderr-only -- stdout (where results/JSON go) is never
// touched, so it can never corrupt piped output even if every other guard
// somehow failed.
package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bashfulrobot/snowstorm/internal/query"
	"github.com/charmbracelet/bubbles/spinner"
	"golang.org/x/term"
)

// shouldShowSpinner reports whether runQueryWithSpinner should animate a
// spinner on stderr: --format table (json's whole point is scripting, so it
// never gets one), stderr a real terminal, and --skip-spinner not set. Pure
// and TTY/flag state passed in explicitly, same reasoning as
// shouldShowPicker -- a non-terminal stderr (redirected/piped/CI) must never
// see spinner output even without --skip-spinner.
func shouldShowSpinner(format string, skipSpinner, stderrIsTTY bool) bool {
	if skipSpinner || !stderrIsTTY {
		return false
	}
	return strings.EqualFold(format, "table")
}

// runQueryWithSpinner runs sqlText via query.Run, optionally animating a
// bubbles/spinner on stderr while it's in flight and clearing it before
// returning. The returned (*query.Result, error) is exactly what a bare
// query.Run(ctx, db, sqlText) call would produce either way -- the spinner
// is pure side effect on stderr, never touching the result or stdout.
func runQueryWithSpinner(ctx context.Context, db *sql.DB, sqlText, format string, skipSpinner bool) (*query.Result, error) {
	if !shouldShowSpinner(format, skipSpinner, term.IsTerminal(int(os.Stderr.Fd()))) {
		return query.Run(ctx, db, sqlText)
	}

	type outcome struct {
		res *query.Result
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, err := query.Run(ctx, db, sqlText)
		done <- outcome{res, err}
	}()

	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	ticker := time.NewTicker(s.Spinner.FPS)
	defer ticker.Stop()

	for {
		select {
		case out := <-done:
			clearSpinnerLine()
			return out.res, out.err
		case <-ticker.C:
			var updated spinner.Model
			updated, _ = s.Update(s.Tick())
			s = updated
			fmt.Fprintf(os.Stderr, "\r%s running query...", s.View())
		}
	}
}

// clearSpinnerLine erases the spinner's line on stderr before results (or an
// error) print, so it never lingers in the terminal or interleaves with
// subsequent output.
func clearSpinnerLine() {
	fmt.Fprint(os.Stderr, "\r\033[K")
}
