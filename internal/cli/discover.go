package cli

import (
	"fmt"
	"regexp"

	"github.com/bashfulrobot/snowstorm/internal/query"
	"github.com/spf13/cobra"
)

var (
	flagDiscoverDatabase string
	flagDiscoverSchema   string
	flagDiscoverTable    string
	flagDiscoverSample   int
	flagDiscoverOut      string
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Explore schema: list tables, or a table's columns (and sample rows)",
	Long: `Schema-exploration helper for finding what to query, built entirely on
the generic query engine (INFORMATION_SCHEMA + SAMPLE under the hood).

  snowstorm discover --database DB --schema SCHEMA               # list tables
  snowstorm discover --database DB --schema SCHEMA --table T      # list columns
  snowstorm discover --database DB --schema SCHEMA --table T --sample 20  # + sample rows

Always prints JSON.`,
	RunE: runDiscover,
}

func init() {
	discoverCmd.Flags().StringVar(&flagDiscoverDatabase, "database", "", "database to inspect (required)")
	discoverCmd.Flags().StringVar(&flagDiscoverSchema, "schema", "", "schema to inspect (required)")
	discoverCmd.Flags().StringVar(&flagDiscoverTable, "table", "", "table to inspect (optional; lists columns instead of tables)")
	discoverCmd.Flags().IntVar(&flagDiscoverSample, "sample", 0, "when --table is set, also fetch this many sample rows")
	discoverCmd.Flags().StringVarP(&flagDiscoverOut, "out", "o", "-", "output path, or - for stdout")
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

	out := map[string]any{
		"database": flagDiscoverDatabase,
		"schema":   flagDiscoverSchema,
	}
	if flagDiscoverTable != "" {
		out["table"] = flagDiscoverTable
		out["columns"] = res
	} else {
		out["tables"] = res
	}

	if flagDiscoverTable != "" && flagDiscoverSample > 0 {
		sampleSQL := fmt.Sprintf(`SELECT * FROM %s.%s.%s SAMPLE (%d ROWS)`,
			flagDiscoverDatabase, flagDiscoverSchema, flagDiscoverTable, flagDiscoverSample)
		sampleRes, err := query.Run(ctx, db, sampleSQL)
		if err != nil {
			return fmt.Errorf("sample rows: %w", err)
		}
		out["sample"] = sampleRes
	}

	w, closer, err := openOutput(flagDiscoverOut)
	if err != nil {
		return err
	}
	defer closer.Close()

	return writeJSON(w, out)
}
