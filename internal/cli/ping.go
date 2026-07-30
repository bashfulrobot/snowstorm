package cli

import (
	"github.com/bashfulrobot/snowstorm/internal/config"
	"github.com/bashfulrobot/snowstorm/internal/query"
	"github.com/spf13/cobra"
)

var (
	flagPingFormat string
	flagPingHuman  bool
	flagPingOut    string
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Verify connectivity and print session/account info",
	Long: `ping opens a Snowflake connection and prints session/account info.
It assumes you're already logged in -- it's a quick connectivity check, not
the command for establishing a session.

If there's no still-valid cached session, ping's underlying connect() will
still trigger auth as a side effect (e.g. a browser popup for
authenticator = "externalbrowser"), but 'snowstorm login' is the intuitive,
dedicated command for that: use it to log in or refresh a session.

Output defaults to a human-readable --format table (--human on by default
too); pass --format json for the old machine-parseable default, byte-for-
byte unchanged. --format, --human, and --connection can also be defaulted
from ~/.snowstorm/config.toml; explicit flags always win.`,
	RunE: runPing,
}

func init() {
	pingCmd.Flags().StringVar(&flagPingFormat, "format", "table", "output format: json or table")
	pingCmd.Flags().BoolVar(&flagPingHuman, "human", true, "table format only: also abbreviate large numbers (5B, 1.2M); commas apply either way")
	pingCmd.Flags().StringVarP(&flagPingOut, "out", "o", "-", "output path, or - for stdout")
	rootCmd.AddCommand(pingCmd)
}

const pingSQL = `
SELECT
  CURRENT_VERSION()   AS snowflake_version,
  CURRENT_ACCOUNT()   AS account,
  CURRENT_USER()      AS user,
  CURRENT_ROLE()      AS role,
  CURRENT_WAREHOUSE() AS warehouse,
  CURRENT_DATABASE()  AS database,
  CURRENT_SCHEMA()    AS schema
`

func runPing(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	flagPingFormat = resolveFormat(cmd.Flags().Changed("format"), flagPingFormat, cfg.Format)
	flagPingHuman = resolveHuman(cmd.Flags().Changed("human"), flagPingHuman, cfg.Human)

	ctx := cmd.Context()
	db, err := connect(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	res, err := query.Run(ctx, db, pingSQL)
	if err != nil {
		return err
	}

	w, closer, err := openOutput(flagPingOut)
	if err != nil {
		return err
	}
	defer closer.Close()

	// groupAccounts=false: ping's "account" column is CURRENT_ACCOUNT() on a
	// single fixed row, not the query command's arbitrary multi-account
	// result sets that findAccountColumn/writeGroupedTable exist for.
	return writeResult(w, res, flagPingFormat, flagPingHuman, false)
}
