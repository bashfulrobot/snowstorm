package cli

import (
	"fmt"
	"os"

	"github.com/bashfulrobot/snowstorm/internal/query"
	"github.com/spf13/cobra"
)

// loginCmd is the obvious, dedicated entry point for establishing or
// refreshing an interactive Snowflake session -- see connect() and
// internal/snow.Connect for what actually triggers auth (a browser window
// for authenticator = "externalbrowser", silently reusing a still-valid
// cached session otherwise).
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Establish or refresh an interactive Snowflake session",
	Long: `login opens a Snowflake connection using the resolved connection
(--connection, $SNOWFLAKE_DEFAULT_CONNECTION_NAME, or config.toml's
connection), which drives whatever auth that connection's entry in
connections.toml specifies. For authenticator = "externalbrowser" this pops
a browser window for interactive SSO when there's no still-valid cached
session, and reuses one silently when there is.

Run this to log in, or to force interactive re-auth once a cached session
has expired -- it's the intuitive command for that, not 'snowstorm ping'.
'ping' assumes you're already logged in: it's a quick connectivity/session
check that happens to also trigger auth as a side effect if you aren't.

See connections.toml's client_store_temporary_credential (externalbrowser)
and client_request_mfa_token (username_password_mfa) for reducing how often
this pops a browser -- README.md documents both.`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

// loginSQL fetches just enough session identity to confirm who/where/as-what
// the login landed as. No warehouse/database/schema -- those are ping's job.
const loginSQL = `
SELECT
  CURRENT_USER()    AS login_user,
  CURRENT_ACCOUNT() AS login_account,
  CURRENT_ROLE()    AS login_role
`

func runLogin(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	db, err := connect(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	res, err := query.Run(ctx, db, loginSQL)
	if err != nil {
		return err
	}

	msg, err := loginConfirmation(res)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, msg)
	return nil
}

// loginConfirmation formats the human-readable confirmation line from
// loginSQL's single-row result -- factored out so it's unit-testable
// without a live Snowflake connection.
func loginConfirmation(res *query.Result) (string, error) {
	if res == nil || len(res.Rows) != 1 {
		n := 0
		if res != nil {
			n = len(res.Rows)
		}
		return "", fmt.Errorf("login: expected exactly one row from session-info query, got %d", n)
	}

	row := res.Rows[0]
	user, _ := row["LOGIN_USER"].(string)
	account, _ := row["LOGIN_ACCOUNT"].(string)
	role, _ := row["LOGIN_ROLE"].(string)

	return fmt.Sprintf("Logged in as %s on account %s (role %s).", user, account, role), nil
}
