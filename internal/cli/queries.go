package cli

import (
	"github.com/spf13/cobra"
)

var queriesCmd = &cobra.Command{
	Use:   "queries",
	Short: "Manage predefined queries (see 'snowstorm query --saved')",
}

var flagQueriesListOut string

var queriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List predefined queries available in --query-dir",
	Long: `List the predefined queries snowstorm query --saved NAME can find:
<name>.sql (plain text) or <name>.toml (name/description/sql fields) files
in --query-dir (default ~/.snowstorm/queries, or $SNOWSTORM_QUERY_DIR).`,
	RunE: runQueriesList,
}

func init() {
	queriesListCmd.Flags().StringVar(&flagQueryDir, "query-dir", "", "directory of predefined queries (default ~/.snowstorm/queries or $SNOWSTORM_QUERY_DIR)")
	queriesListCmd.Flags().StringVarP(&flagQueriesListOut, "out", "o", "-", "output path, or - for stdout")
	queriesCmd.AddCommand(queriesListCmd)
	rootCmd.AddCommand(queriesCmd)
}

func runQueriesList(cmd *cobra.Command, args []string) error {
	items, err := listSavedQueries()
	if err != nil {
		return err
	}

	w, closer, err := openOutput(flagQueriesListOut)
	if err != nil {
		return err
	}
	defer closer.Close()

	return writeJSON(w, map[string]any{
		"query_dir": queryDir(),
		"queries":   items,
	})
}
