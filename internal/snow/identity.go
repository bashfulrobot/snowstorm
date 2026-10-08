package snow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/bashfulrobot/snowstorm/internal/gosnowflake"
)

// tokenIdentity resolves the host and user gosnowflake keys the cached token
// by, from the named connection in connections.toml. Only the host, account
// and user fields are read; nothing else from the file is kept.
func tokenIdentity(opts Options) (host, user string, err error) {
	home := opts.Home
	if home == "" {
		home = os.Getenv(envHome)
	}
	if home == "" {
		h, herr := os.UserHomeDir()
		if herr != nil {
			return "", "", errors.New("cannot locate home directory")
		}
		home = filepath.Join(h, ".snowflake")
	}
	name := opts.ConnectionName
	if name == "" {
		name = os.Getenv(envConnectionName)
	}
	if name == "" {
		name = "default"
	}

	var doc map[string]any
	path := filepath.Join(home, "connections.toml")
	if _, derr := toml.DecodeFile(path, &doc); derr != nil {
		return "", "", fmt.Errorf("cannot read %s", path)
	}
	tbl, _ := doc[name].(map[string]any)
	if tbl == nil { // also accept the nested [connections.<name>] shape
		if nested, ok := doc["connections"].(map[string]any); ok {
			tbl, _ = nested[name].(map[string]any)
		}
	}
	if tbl == nil {
		return "", "", fmt.Errorf("no connection %q in %s", name, path)
	}
	str := func(k string) string { v, _ := tbl[k].(string); return strings.TrimSpace(v) }

	user = str("user")
	host = str("host")
	if host == "" {
		if acct := str("account"); acct != "" {
			host = acct + ".snowflakecomputing.com"
		}
	}
	if host == "" || user == "" {
		return "", "", fmt.Errorf("connection %q has no user and host/account", name)
	}
	return host, user, nil
}

// ClearCachedToken asks gosnowflake's OS-default credential storage (macOS
// login keychain, Linux file cache, Windows Credential Manager) to delete the
// cached SSO ID token for the resolved connection. The driver does not report
// whether an entry existed. It returns the host and user it acted on.
func ClearCachedToken(opts Options) (host, user string, err error) {
	host, user, err = tokenIdentity(opts)
	if err != nil {
		return "", "", err
	}
	gosnowflake.ClearIDToken(host, user)
	return host, user, nil
}
