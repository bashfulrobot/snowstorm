package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bashfulrobot/snowstorm/internal/query"
	"github.com/spf13/cobra"
)

var (
	flagQueryFile   string
	flagQuerySaved  string
	flagQueryDir    string
	flagQueryFormat string
	flagQueryOut    string
	flagQueryHuman  bool
)

var queryCmd = &cobra.Command{
	Use:   "query [SQL]",
	Short: "Run a SQL statement and print the result as JSON",
	Long: `Run a SQL statement against Snowflake and print the result set.

The statement can come from four places, in this priority order:
  1. the SQL positional argument
  2. --file path/to/query.sql        (a one-off file, any path)
  3. --saved NAME                    (a predefined query by name; see 'snowstorm queries list')
  4. stdin (piped in, e.g. cat query.sql | snowstorm query)

Predefined queries live in --query-dir (default ~/.snowstorm/queries, or
$SNOWSTORM_QUERY_DIR) as either <name>.sql (plain text) or <name>.toml
(structured: name/description/sql fields) -- see 'snowstorm queries list'.

Output defaults to JSON: {"columns": [...], "rows": [{...}, ...], "row_count": N}.
Use --format table for a human-readable view: numbers of 1,000+ are always
comma-grouped there (12,345,678), exact, no flag needed. Add --human to
additionally abbreviate numbers of 1,000,000+ with a K/M/B/T suffix (5B,
1.2M) for a quicker scan -- that rounds, so it's opt-in. JSON is always
exact either way. A column named ACCOUNT/ACCOUNT_NAME is grouped instead of
repeated on every row, in both --format table and JSON.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runQuery,
}

func init() {
	queryCmd.Flags().StringVarP(&flagQueryFile, "file", "f", "", "read the SQL statement from this file")
	queryCmd.Flags().StringVarP(&flagQuerySaved, "saved", "s", "", "run a predefined query by name from --query-dir")
	queryCmd.Flags().StringVar(&flagQueryDir, "query-dir", "", "directory of predefined queries (default ~/.snowstorm/queries or $SNOWSTORM_QUERY_DIR)")
	queryCmd.Flags().StringVar(&flagQueryFormat, "format", "json", "output format: json or table")
	queryCmd.Flags().BoolVar(&flagQueryHuman, "human", false, "table format only: also abbreviate large numbers (5B, 1.2M); commas apply either way")
	queryCmd.Flags().StringVarP(&flagQueryOut, "out", "o", "-", "output path, or - for stdout")
	rootCmd.AddCommand(queryCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
	sqlText, err := resolveSQL(args)
	if err != nil {
		return err
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

	res, err := query.Run(ctx, db, sqlText)
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
