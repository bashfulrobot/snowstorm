package cli

import (
	"fmt"

	"github.com/bashfulrobot/snowstorm/internal/credcache"
	"github.com/spf13/cobra"
)

// logoutCmd forgets the file-cached SSO token. It never contacts Snowflake.
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Delete the file-cached SSO token",
	Long: `logout deletes snowstorm's file-based token cache
(credential_store = "file"; ~/Library/Caches/snowstorm on macOS), so the next
command opens the browser again. It does not contact Snowflake and does not
touch the login keychain or connections.toml.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := credcache.Clear(); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "cached SSO token removed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
