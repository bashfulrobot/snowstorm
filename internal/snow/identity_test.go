package snow

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConns(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "connections.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestTokenIdentity(t *testing.T) {
	t.Setenv(envConnectionName, "")
	t.Setenv(envHome, "")
	home := writeConns(t, `
[a]
account = "xy123.us-east-1"
user = "me@example.com"
password = "never-read"

[b]
host = "custom.example.com"
user = "other"

[connections.c]
account = "nested"
user = "n"
`)
	cases := []struct {
		name, host, user string
		wantErr          bool
	}{
		{"a", "xy123.us-east-1.snowflakecomputing.com", "me@example.com", false},
		{"b", "custom.example.com", "other", false},
		{"c", "nested.snowflakecomputing.com", "n", false},
		{"missing", "", "", true},
	}
	for _, c := range cases {
		h, u, err := tokenIdentity(Options{ConnectionName: c.name, Home: home})
		if (err != nil) != c.wantErr || h != c.host || u != c.user {
			t.Errorf("%s: got %q %q %v", c.name, h, u, err)
		}
	}
	if _, _, err := tokenIdentity(Options{ConnectionName: "a", Home: t.TempDir()}); err == nil {
		t.Error("missing connections.toml accepted")
	}
	t.Setenv(envConnectionName, "b")
	if h, _, err := tokenIdentity(Options{Home: home}); err != nil || h != "custom.example.com" {
		t.Errorf("env connection name not honoured: %q %v", h, err)
	}
}
