package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bashfulrobot/snowstorm/internal/config"
	"github.com/spf13/cobra"
)

var (
	flagQueryFile        string
	flagQuerySaved       string
	flagQueryDir         string
	flagQueryFormat      string
	flagQueryOut         string
	flagQueryHuman       bool
	flagQuerySkipPick    bool
	flagQuerySkipSpinner bool
)

var queryCmd = &cobra.Command{
	Use:   "query [SQL]",
	Short: "Run a SQL statement and print the result",
	Long: `Run a SQL statement against Snowflake and print the result set.

The statement can come from four places, in this priority order:
  1. the SQL positional argument
  2. --file path/to/query.sql        (a one-off file, any path)
  3. --saved NAME                    (a predefined query by name; see 'snowstorm queries list')
  4. an interactive picker over saved queries, or stdin -- see below

Predefined queries live in --query-dir (default ~/.snowstorm/queries, or
$SNOWSTORM_QUERY_DIR) as either <name>.sql (plain text) or <name>.toml
(structured: name/description/sql fields) -- see 'snowstorm queries list'.

Bare invocations are tuned for a human at a terminal: output defaults to a
readable --format table (numbers of 1,000+ are always comma-grouped there,
12,345,678, exact, no flag needed; --human additionally abbreviates
1,000,000+ with a K/M/B/T suffix like 5B or 1.2M -- that rounds, so it's
opt-in and on by default). Use --format json for the old machine-parseable
default: {"columns": [...], "rows": [{...}, ...], "row_count": N}, always
byte-for-byte the same regardless of --human. A column named
ACCOUNT/ACCOUNT_NAME is grouped instead of repeated on every row, in both
--format table and JSON.

When none of [SQL, --file, --saved] is given and both stdin and stdout are
real terminals, an fzf-style fuzzy picker over --query-dir's saved queries
opens instead of reading stdin; Enter runs the chosen query, Ctrl-C/Esc
exits cleanly with no output. Pass --skip-pick, or run non-interactively
(piped stdin/stdout), to fall back to the old behavior of reading stdin.

A spinner shows on stderr while a --format table query runs, when stderr is
a real terminal; --skip-spinner or --format json turns it off. Nothing
interactive ever triggers for a non-terminal caller, regardless of flags --
these are courtesy escape hatches on top of that TTY check, not the only
guard.

--format, --human, and --query-dir can also be defaulted from
~/.snowstorm/config.toml so they don't need to be passed every time;
explicit flags always win over it.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runQuery,
}

func init() {
	queryCmd.Flags().StringVarP(&flagQueryFile, "file", "f", "", "read the SQL statement from this file")
	queryCmd.Flags().StringVarP(&flagQuerySaved, "saved", "s", "", "run a predefined query by name from --query-dir")
	queryCmd.Flags().StringVar(&flagQueryDir, "query-dir", "", "directory of predefined queries (default ~/.snowstorm/queries or $SNOWSTORM_QUERY_DIR)")
	queryCmd.Flags().StringVar(&flagQueryFormat, "format", "table", "output format: json or table")
	queryCmd.Flags().BoolVar(&flagQueryHuman, "human", true, "table format only: also abbreviate large numbers (5B, 1.2M); commas apply either way")
	queryCmd.Flags().StringVarP(&flagQueryOut, "out", "o", "-", "output path, or - for stdout")
	queryCmd.Flags().BoolVar(&flagQuerySkipPick, "skip-pick", false, "never show the interactive saved-query picker; read stdin instead (agent/script escape hatch)")
	queryCmd.Flags().BoolVar(&flagQuerySkipSpinner, "skip-spinner", false, "never show the stderr progress spinner while a query runs")
	rootCmd.AddCommand(queryCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	flagQueryFormat = resolveFormat(cmd.Flags().Changed("format"), flagQueryFormat, cfg.Format)
	flagQueryHuman = resolveHuman(cmd.Flags().Changed("human"), flagQueryHuman, cfg.Human)
	flagQueryDir = resolveQueryDir(cmd.Flags().Changed("query-dir"), flagQueryDir, os.Getenv("SNOWSTORM_QUERY_DIR"), cfg.QueryDir)

	sqlText, cancelled, err := resolveSQLOrPick(args)
	if err != nil {
		return err
	}
	if cancelled {
		// Ctrl-C/Esc in the picker, or a picker shown over an empty query
		// dir: a clean, deliberate no-op, not an error -- no output, exit 0.
		return nil
	}
	if strings.TrimSpace(sqlText) == "" {
		return fmt.Errorf("no SQL provided (pass it as an argument, --file, --saved, or via stdin)")
	}

	ctx := cmd.Context()
	db, err := connect(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	res, err := runQueryWithSpinner(ctx, db, sqlText, flagQueryFormat, flagQuerySkipSpinner)
	if err != nil {
		return err
	}

	w, closer, err := openOutput(flagQueryOut)
	if err != nil {
		return err
	}
	defer closer.Close()

	return writeResult(w, res, flagQueryFormat, flagQueryHuman, true)
}

// resolveSQL applies the SQL-source priority: arg > --file > --saved > stdin.
func resolveSQL(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	if flagQueryFile != "" {
		b, err := os.ReadFile(flagQueryFile)
		if err != nil {
			return "", fmt.Errorf("read --file %q: %w", flagQueryFile, err)
		}
		return string(b), nil
	}
	if flagQuerySaved != "" {
		return resolveSavedQuery(flagQuerySaved)
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return string(b), nil
}

// resolveFormat applies the --format priority: explicit flag > config.toml's
// format > the built-in default ("table" -- bare invocations are tuned for a
// human at a terminal; agents/scripts opt into the old machine-parseable
// behavior with an explicit --format json). There's no env var tier here.
func resolveFormat(flagChanged bool, flagValue, configValue string) string {
	if flagChanged {
		return flagValue
	}
	if configValue != "" {
		return configValue
	}
	return "table"
}

// resolveHuman applies the --human priority: explicit flag > config.toml's
// human > the built-in default (true, for the same "humans get the nice
// output by default" reason as resolveFormat). --human is a bool flag, so
// its Changed() state must be checked explicitly -- the zero value false is
// indistinguishable from an explicit --human=false.
//
// configValue can't express "config.toml doesn't mention human" separately
// from "config.toml says human = false" -- config.Config.Human is a plain
// bool, not a pointer (internal/config is out of scope for this change), so
// both decode to false. That was harmless while the built-in default was
// also false; now that the default is true, it means config.toml can ask
// for human=true (redundant with the new default) but can no longer turn it
// off -- only an explicit --human=false flag can. Documented trade-off, not
// an oversight.
func resolveHuman(flagChanged, flagValue, configValue bool) bool {
	if flagChanged {
		return flagValue
	}
	_ = configValue // see doc comment: can't distinguish "unset" from "false"
	return true
}

// resolveQueryDir applies the --query-dir priority: explicit flag >
// $SNOWSTORM_QUERY_DIR > config.toml's query_dir > "" (queryDir(), in
// savedquery.go, falls back to ~/.snowstorm/queries when it sees "").
func resolveQueryDir(flagChanged bool, flagValue, envValue, configValue string) string {
	if flagChanged {
		return flagValue
	}
	if envValue != "" {
		return envValue
	}
	return configValue
}
