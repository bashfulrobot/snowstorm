package cli

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/bashfulrobot/snowstorm/internal/config"
	"github.com/bashfulrobot/snowstorm/internal/query"
	"github.com/spf13/cobra"
)

var (
	flagDiscoverDatabase string
	flagDiscoverSchema   string
	flagDiscoverTable    string
	flagDiscoverSample   int
	flagDiscoverOut      string
	flagDiscoverFormat   string
	flagDiscoverHuman    bool
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Explore schema: list tables, or a table's columns (and sample rows)",
	Long: `Schema-exploration helper for finding what to query, built entirely on
the generic query engine (INFORMATION_SCHEMA + SAMPLE under the hood).

  snowstorm discover --database DB --schema SCHEMA               # list tables
  snowstorm discover --database DB --schema SCHEMA --table T      # list columns
  snowstorm discover --database DB --schema SCHEMA --table T --sample 20  # + sample rows

Output defaults to a human-readable --format table: a "database: X, schema:
Y[, table: Z]" header line above each sub-table (tables, or columns and an
optional sample), the same grouped-table style 'snowstorm query' uses for
per-account groups. Pass --format json for the old machine-parseable shape,
byte-for-byte unchanged: {"database", "schema", "tables"} or {"database",
"schema", "table", "columns", "sample"?}. --format and --human can also be
defaulted from ~/.snowstorm/config.toml; explicit flags always win.`,
	RunE: runDiscover,
}

func init() {
	discoverCmd.Flags().StringVar(&flagDiscoverDatabase, "database", "", "database to inspect (required)")
	discoverCmd.Flags().StringVar(&flagDiscoverSchema, "schema", "", "schema to inspect (required)")
	discoverCmd.Flags().StringVar(&flagDiscoverTable, "table", "", "table to inspect (optional; lists columns instead of tables)")
	discoverCmd.Flags().IntVar(&flagDiscoverSample, "sample", 0, "when --table is set, also fetch this many sample rows")
	discoverCmd.Flags().StringVarP(&flagDiscoverOut, "out", "o", "-", "output path, or - for stdout")
	discoverCmd.Flags().StringVar(&flagDiscoverFormat, "format", "table", "output format: json or table")
	discoverCmd.Flags().BoolVar(&flagDiscoverHuman, "human", true, "table format only: also abbreviate large numbers; commas apply either way")
	_ = discoverCmd.MarkFlagRequired("database")
	_ = discoverCmd.MarkFlagRequired("schema")
	rootCmd.AddCommand(discoverCmd)
}

// validIdentifier matches an unquoted Snowflake identifier. Database/schema/
// table names are interpolated into SQL (Snowflake has no way to bind
// identifiers as query parameters), so every one is checked against this
// before it touches a query string.
var validIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*$`)

func checkIdentifier(kind, name string) error {
	if !validIdentifier.MatchString(name) {
		return fmt.Errorf("invalid %s %q: must be a plain identifier ([A-Za-z_][A-Za-z0-9_$]*)", kind, name)
	}
	return nil
}

func runDiscover(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	flagDiscoverFormat = resolveFormat(cmd.Flags().Changed("format"), flagDiscoverFormat, cfg.Format)
	flagDiscoverHuman = resolveHuman(cmd.Flags().Changed("human"), flagDiscoverHuman, cfg.Human)

	if err := checkIdentifier("database", flagDiscoverDatabase); err != nil {
		return err
	}
	if err := checkIdentifier("schema", flagDiscoverSchema); err != nil {
		return err
	}
	if flagDiscoverTable != "" {
		if err := checkIdentifier("table", flagDiscoverTable); err != nil {
			return err
		}
	}

	ctx := cmd.Context()
	db, err := connect(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	var sqlText string
	switch {
	case flagDiscoverTable != "":
		sqlText = fmt.Sprintf(`
SELECT column_name, data_type, is_nullable, ordinal_position, comment
FROM %s.INFORMATION_SCHEMA.COLUMNS
WHERE table_schema = '%s' AND table_name = '%s'
ORDER BY ordinal_position`,
			flagDiscoverDatabase, flagDiscoverSchema, flagDiscoverTable)
	default:
		sqlText = fmt.Sprintf(`
SELECT table_name, table_type, row_count, bytes, comment
FROM %s.INFORMATION_SCHEMA.TABLES
WHERE table_schema = '%s'
ORDER BY table_name`,
			flagDiscoverDatabase, flagDiscoverSchema)
	}

	res, err := query.Run(ctx, db, sqlText)
	if err != nil {
		return err
	}

	var sampleRes *query.Result
	if flagDiscoverTable != "" && flagDiscoverSample > 0 {
		sampleSQL := fmt.Sprintf(`SELECT * FROM %s.%s.%s SAMPLE (%d ROWS)`,
			flagDiscoverDatabase, flagDiscoverSchema, flagDiscoverTable, flagDiscoverSample)
		sampleRes, err = query.Run(ctx, db, sampleSQL)
		if err != nil {
			return fmt.Errorf("sample rows: %w", err)
		}
	}

	w, closer, err := openOutput(flagDiscoverOut)
	if err != nil {
		return err
	}
	defer closer.Close()

	switch strings.ToLower(flagDiscoverFormat) {
	case "", "json":
		out := map[string]any{
			"database": flagDiscoverDatabase,
			"schema":   flagDiscoverSchema,
		}
		if flagDiscoverTable != "" {
			out["table"] = flagDiscoverTable
			out["columns"] = res
			if sampleRes != nil {
				out["sample"] = sampleRes
			}
		} else {
			out["tables"] = res
		}
		return writeJSON(w, out)
	case "table":
		return writeDiscoverTable(w, res, sampleRes, flagDiscoverHuman)
	default:
		return fmt.Errorf("unknown --format %q (want json or table)", flagDiscoverFormat)
	}
}

// writeDiscoverTable renders discover's --format table output: a header line
// naming what's being shown, then a flat table per query.Result, in the same
// "header line above a sub-table" style writeGroupedTable (output.go) uses
// for per-account groups. flagDiscoverTable/-Database/-Schema/-Sample are
// read directly (package-level flag vars, same as the rest of this file)
// rather than threaded through as parameters.
func writeDiscoverTable(w io.Writer, res, sampleRes *query.Result, human bool) error {
	if flagDiscoverTable == "" {
		fmt.Fprintf(w, "database: %s, schema: %s\n", flagDiscoverDatabase, flagDiscoverSchema)
		return writeFlatTable(w, res.Columns, res.Rows, human, res.RowCount)
	}

	fmt.Fprintf(w, "database: %s, schema: %s, table: %s\n", flagDiscoverDatabase, flagDiscoverSchema, flagDiscoverTable)
	if err := writeFlatTable(w, res.Columns, res.Rows, human, res.RowCount); err != nil {
		return err
	}

	if sampleRes != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "sample (%d row(s) requested):\n", flagDiscoverSample)
		if err := writeFlatTable(w, sampleRes.Columns, sampleRes.Rows, human, sampleRes.RowCount); err != nil {
			return err
		}
	}
	return nil
}
