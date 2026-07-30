// Command snowstorm is a Snowflake data-access CLI: connect via
// connections.toml, run SQL, get structured JSON back.
package main

import (
	"os"

	"github.com/bashfulrobot/snowstorm/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		cli.PrintError(err)
		os.Exit(1)
	}
}
