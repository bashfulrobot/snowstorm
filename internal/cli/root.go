// Package cli implements snowstorm's command surface: a small, composable
// Snowflake data-access CLI. Every command that talks to Snowflake goes
// through internal/snow.Connect (connections.toml) and internal/query.Run
// (SQL -> JSON-safe rows); commands are thin wrappers that pick input/output.
package cli

import (
	"os"
	"time"

	"github.com/bashfulrobot/snowstorm/internal/config"
	"github.com/spf13/cobra"
)

var (
	flagConnection string
	flagHome       string
	flagTimeout    time.Duration
)

// envConnectionName is the env var gosnowflake's own connections.toml
// loader reads (see internal/snow's envConnectionName). Read explicitly
// here -- rather than through internal/snow -- so an already-exported env
// var can outrank config.toml without config.toml ever touching it.
const envConnectionName = "SNOWFLAKE_DEFAULT_CONNECTION_NAME"

// rootCmd is the base command; all others are registered on it in init().
var rootCmd = &cobra.Command{
	Use:   "snowstorm",
	Short: "Snowflake data-access CLI: run queries, get structured JSON back",
	Long: `snowstorm connects to Snowflake using a named connection from
~/.snowflake/connections.toml (the same file the Snowflake CLI/connector
tooling reads) and runs SQL against it, returning structured JSON.

It is a data-access tool, not a reporting tool: it makes Snowflake data
easy to pull into scripts and other tools via plain JSON on stdout.

An optional ~/.snowstorm/config.toml can set defaults for --connection,
--format, --human, and --query-dir so they don't need to be passed on
every invocation; explicit flags always win. All fields are optional:

  connection = "kong-revops"
  format     = "table"
  human      = true
  query_dir  = "/custom/path/to/queries"`,
	SilenceUsage: true,
	// SilenceErrors: cobra's own error printer is turned off tool-wide so
	// PrintError (errstyle.go) is the only thing that ever writes a command
	// error to stderr -- otherwise every error would print twice (cobra's
	// "Error: ..." plus main.go's own "snowstorm: ..." line).
	SilenceErrors: true,
	// PersistentPreRunE runs before every command's RunE (commands below
	// don't define their own, so this one applies tool-wide) and resolves
	// --connection against config.toml before connect() (see connect.go)
	// ever reads flagConnection.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		flagConnection = resolveConnection(
			cmd.Flags().Changed("connection"),
			flagConnection,
			os.Getenv(envConnectionName),
			cfg.Connection,
		)
		return nil
	},
}

// resolveConnection applies the --connection priority: the flag itself (if
// the user passed it) > an already-exported $SNOWFLAKE_DEFAULT_CONNECTION_NAME
// (never overridden by config.toml -- it's already governing gosnowflake's
// own connections.toml lookup) > config.toml's connection > "" (today's
// default: let connections.toml's own default connection apply).
func resolveConnection(flagChanged bool, flagValue, envValue, configValue string) string {
	if flagChanged {
		return flagValue
	}
	if envValue != "" {
		return envValue
	}
	return configValue
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagConnection, "connection", "c", "",
		`named connection from ~/.snowflake/connections.toml (default: that file's own default connection)`)
	rootCmd.PersistentFlags().StringVar(&flagHome, "home", "",
		`override directory containing connections.toml (default: ~/.snowflake)`)
	rootCmd.PersistentFlags().DurationVar(&flagTimeout, "timeout", 30*time.Second,
		`timeout for the initial connection check (does not bound query execution)`)
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false,
		`disable the colorized "Error:" prefix on command errors (also respects the NO_COLOR env var)`)
}

// Execute runs the root command; call this from main().
func Execute() error {
	return rootCmd.Execute()
}
