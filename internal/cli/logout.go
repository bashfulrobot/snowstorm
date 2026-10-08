package cli

import (
	"fmt"
	"runtime"

	"github.com/bashfulrobot/snowstorm/internal/credcache"
	"github.com/bashfulrobot/snowstorm/internal/snow"
	"github.com/spf13/cobra"
)

// logoutCmd forgets the locally cached SSO token. It never contacts Snowflake.
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Delete the locally cached SSO token",
	Long: `logout deletes the locally cached SSO token so the next command opens the
browser again. It never contacts Snowflake: the server-side session is NOT
revoked and stays valid until it expires.

Where it removes the token from depends on the credential store:
  - macOS default (file cache): snowstorm's cache file in the user cache dir.
  - macOS with credential_store = "keychain", Linux, Windows: the driver's own
    storage (login keychain, ~/.cache/snowflake, Credential Manager) for the
    resolved --connection. Removing a keychain item may show a keychain prompt.
  - In file mode it also asks the login keychain to drop a stale item from
    before the file cache existed.
The driver does not report whether an item existed, so logout says what it
asked for, not what it found. It does not touch connections.toml.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := snow.Options{ConnectionName: flagConnection, Home: flagHome}
		var fileDir string
		var fileErr error
		fileMode := runtime.GOOS == "darwin" && resolvedCredentialStore == credcache.ModeFile
		if fileMode {
			if fileDir, fileErr = credcache.DefaultDir(); fileErr == nil {
				fileErr = credcache.Clear()
			}
		}
		host, user, drvErr := snow.ClearCachedToken(opts)
		for _, l := range logoutReport(runtime.GOOS, resolvedCredentialStore, fileDir, fileErr, host, user, drvErr) {
			fmt.Fprintln(cmd.OutOrStdout(), l)
		}
		return fileErr
	},
}

// storeLabel names the driver's own storage on goos.
func storeLabel(goos string) string {
	switch goos {
	case "darwin":
		return "login keychain"
	case "windows":
		return "Windows Credential Manager"
	}
	return "the driver's file cache (~/.cache/snowflake)"
}

// logoutReport builds the honest outcome lines: what was removed from where,
// what was only requested, and what could not be done.
func logoutReport(goos, mode, fileDir string, fileErr error, host, user string, drvErr error) []string {
	var out []string
	label := storeLabel(goos)
	fileMode := goos == "darwin" && mode == credcache.ModeFile
	if fileMode {
		if fileErr != nil {
			out = append(out, "could not remove snowstorm's file cache: "+fileErr.Error())
		} else {
			out = append(out, "removed snowstorm's file-cached token from "+fileDir)
		}
		label = "login keychain (stale pre-file-cache item)"
	} else {
		out = append(out, fmt.Sprintf("the token is kept in the %s, not in snowstorm's file cache; the file cache was not touched", storeLabel(goos)))
	}
	if drvErr != nil {
		out = append(out, fmt.Sprintf("could not resolve the connection's host/user (%v); nothing was removed from the %s", drvErr, label))
	} else {
		out = append(out, fmt.Sprintf("asked the %s to delete the token for %s at %s (the driver does not report whether one existed)", label, user, host))
	}
	out = append(out, "the Snowflake server-side session was not revoked")
	return out
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
