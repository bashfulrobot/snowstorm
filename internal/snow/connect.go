// Package snow wraps Snowflake connection setup on top of gosnowflake's
// native connections.toml support (~/.snowflake/connections.toml).
package snow

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/snowflakedb/gosnowflake/v2" // registers the "snowflake" database/sql driver
)

// Env vars read by gosnowflake's internal connections.toml loader.
const (
	envConnectionName = "SNOWFLAKE_DEFAULT_CONNECTION_NAME"
	envHome           = "SNOWFLAKE_HOME"

	// autoConfigDSN is the magic DSN string gosnowflake recognizes to load
	// the named connection from connections.toml instead of parsing a DSN.
	autoConfigDSN = "autoConfig"
)

// Options configures how Connect resolves and opens a Snowflake connection.
type Options struct {
	// ConnectionName selects a named entry from connections.toml
	// ([connections.<name>]). Empty means gosnowflake's own default ("default").
	ConnectionName string

	// Home overrides the directory containing connections.toml.
	// Empty means gosnowflake's own default (~/.snowflake).
	Home string

	// PingTimeout bounds the initial connectivity check in Connect.
	PingTimeout time.Duration
}

// Connect opens a *sql.DB against Snowflake using a named connection from
// connections.toml, and verifies connectivity with a bounded PingContext.
//
// Auth is whatever the named connection specifies (e.g. authenticator =
// "externalbrowser" opens a browser window for interactive SSO) -- this is
// meant to be run interactively, not from a headless/unattended context.
// "snowstorm login" is the dedicated command for triggering this explicitly.
//
// connections.toml is user-owned config outside this package's control, but
// two of its fields are worth knowing about if browser popups happen more
// often than expected: client_store_temporary_credential (caches the SSO id
// token for authenticator = "externalbrowser") and client_request_mfa_token
// (same idea, for authenticator = "username_password_mfa"). Both default to
// on for Windows/macOS but off for Linux (gosnowflake v2.1.0), and on Linux
// need no keyring daemon -- just a plain file cache under
// ~/.cache/snowflake (see README.md). Neither controls how long the cached
// token stays valid; that's governed entirely by the Snowflake account's
// server-side auth/session policies, not anything client-side.
func Connect(ctx context.Context, opts Options) (*sql.DB, error) {
	if opts.ConnectionName != "" {
		if err := os.Setenv(envConnectionName, opts.ConnectionName); err != nil {
			return nil, fmt.Errorf("snow: set %s: %w", envConnectionName, err)
		}
	}
	if opts.Home != "" {
		if err := os.Setenv(envHome, opts.Home); err != nil {
			return nil, fmt.Errorf("snow: set %s: %w", envHome, err)
		}
	}

	db, err := sql.Open("snowflake", autoConfigDSN)
	if err != nil {
		return nil, fmt.Errorf("snow: open connection %q: %w", connectionLabel(opts.ConnectionName), err)
	}

	timeout := opts.PingTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("snow: connect %q: %w", connectionLabel(opts.ConnectionName), err)
	}

	return db, nil
}

func connectionLabel(name string) string {
	if name == "" {
		return "default"
	}
	return name
}
