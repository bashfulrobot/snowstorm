// Package cli implements snowstorm's command surface: a small, composable
// Snowflake data-access CLI. Every command that talks to Snowflake goes
// through internal/snow.Connect (connections.toml) and internal/query.Run
// (SQL -> JSON-safe rows); commands are thin wrappers that pick input/output.
package cli

import (
	"time"

	"github.com/spf13/cobra"
)

var (
	flagConnection string
	flagHome       string
	flagTimeout    time.Duration
)

// rootCmd is the base command; all others are registered on it in init().
var rootCmd = &cobra.Command{
	Use:   "snowstorm",
	Short: "Snowflake data-access CLI: run queries, get structured JSON back",
	Long: `snowstorm connects to Snowflake using a named connection from
~/.snowflake/connections.toml (the same file the Snowflake CLI/connector
tooling reads) and runs SQL against it, returning structured JSON.

It is a data-access tool, not a reporting tool: it makes Snowflake data
easy to pull into scripts and other tools via plain JSON on stdout.`,
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagConnection, "connection", "c", "",
		`named connection from ~/.snowflake/connections.toml (default: that file's own default connection)`)
	rootCmd.PersistentFlags().StringVar(&flagHome, "home", "",
		`override directory containing connections.toml (default: ~/.snowflake)`)
	rootCmd.PersistentFlags().DurationVar(&flagTimeout, "timeout", 30*time.Second,
		`timeout for the initial connection check (does not bound query execution)`)
}

// Execute runs the root command; call this from main().
func Execute() error {
	return rootCmd.Execute()
}
