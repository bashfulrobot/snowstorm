package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bashfulrobot/snowstorm/internal/config"
	"github.com/spf13/cobra"
)

var queriesCmd = &cobra.Command{
	Use:   "queries",
	Short: "Manage predefined queries (see 'snowstorm query --saved')",
}

var (
	flagQueriesListOut    string
	flagQueriesListFormat string
	flagQueriesListHuman  bool
)

var queriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List predefined queries available in --query-dir",
	Long: `List the predefined queries snowstorm query --saved NAME can find:
<name>.sql (plain text) or <name>.toml (name/description/sql fields) files
in --query-dir (default ~/.snowstorm/queries, or $SNOWSTORM_QUERY_DIR).

Output defaults to a human-readable --format table of name/kind/description;
pass --format json for the old machine-parseable shape (byte-for-byte
unchanged): {"query_dir": ..., "queries": [{"name", "kind", "path",
"description"}, ...]}. --format and --human can also be defaulted from
~/.snowstorm/config.toml; explicit flags always win.`,
	RunE: runQueriesList,
}

func init() {
	queriesListCmd.Flags().StringVar(&flagQueryDir, "query-dir", "", "directory of predefined queries (default ~/.snowstorm/queries or $SNOWSTORM_QUERY_DIR)")
	queriesListCmd.Flags().StringVar(&flagQueriesListFormat, "format", "table", "output format: json or table")
	queriesListCmd.Flags().BoolVar(&flagQueriesListHuman, "human", true, "table format only: also abbreviate large numbers; commas apply either way")
	queriesListCmd.Flags().StringVarP(&flagQueriesListOut, "out", "o", "-", "output path, or - for stdout")
	queriesCmd.AddCommand(queriesListCmd)
	rootCmd.AddCommand(queriesCmd)
}

func runQueriesList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	flagQueriesListFormat = resolveFormat(cmd.Flags().Changed("format"), flagQueriesListFormat, cfg.Format)
	flagQueriesListHuman = resolveHuman(cmd.Flags().Changed("human"), flagQueriesListHuman, cfg.Human)
	flagQueryDir = resolveQueryDir(cmd.Flags().Changed("query-dir"), flagQueryDir, os.Getenv("SNOWSTORM_QUERY_DIR"), cfg.QueryDir)

	items, err := listSavedQueries()
	if err != nil {
		return err
	}

	w, closer, err := openOutput(flagQueriesListOut)
	if err != nil {
		return err
	}
	defer closer.Close()

	switch strings.ToLower(flagQueriesListFormat) {
	case "", "json":
		return writeJSON(w, map[string]any{
			"query_dir": queryDir(),
			"queries":   items,
		})
	case "table":
		return writeSavedQueriesTable(w, items, flagQueriesListHuman)
	default:
		return fmt.Errorf("unknown --format %q (want json or table)", flagQueriesListFormat)
	}
}

// savedQueriesTableColumns are the columns shown by `queries list --format
// table` -- deliberately narrower than the JSON shape (no Path) since a
// human scanning the list cares about name/kind/description, not the file
// path backing each entry.
var savedQueriesTableColumns = []string{"NAME", "KIND", "DESCRIPTION"}

func savedQueriesRows(items []savedQueryInfo) []map[string]any {
	rows := make([]map[string]any, len(items))
	for i, it := range items {
		rows[i] = map[string]any{
			"NAME":        it.Name,
			"KIND":        it.Kind,
			"DESCRIPTION": it.Description,
		}
	}
	return rows
}

func writeSavedQueriesTable(w io.Writer, items []savedQueryInfo, human bool) error {
	fmt.Fprintf(w, "query_dir: %s\n\n", queryDir())
	return writeFlatTable(w, savedQueriesTableColumns, savedQueriesRows(items), human, len(items))
}
