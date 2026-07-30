package cli

import (
	"github.com/bashfulrobot/snowstorm/internal/query"
	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Verify connectivity and print session/account info as JSON",
	Long: `ping opens a Snowflake connection and prints session/account info
as JSON. It assumes you're already logged in -- it's a quick connectivity
check, not the command for establishing a session.

If there's no still-valid cached session, ping's underlying connect() will
still trigger auth as a side effect (e.g. a browser popup for
authenticator = "externalbrowser"), but 'snowstorm login' is the intuitive,
dedicated command for that: use it to log in or refresh a session.`,
	RunE: runPing,
}

func init() {
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

	w, closer, err := openOutput("-")
	if err != nil {
		return err
	}
	defer closer.Close()

	return writeResult(w, res, "json", false, false)
}
