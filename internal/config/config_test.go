package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.toml")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("missing file: expected no error, got %v", err)
	}
	if cfg != (Config{}) {
		t.Fatalf("missing file: expected zero-value Config, got %+v", cfg)
	}
}

func TestLoadFromMalformed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("connection = [this is not valid toml"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadFrom(path); err == nil {
		t.Fatal("malformed TOML: expected an error, got nil")
	}
}

func TestLoadFromValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
connection = "kong-revops"
format     = "table"
human      = true
query_dir  = "/custom/path/to/queries"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("valid file: unexpected error: %v", err)
	}
	want := Config{
		Connection: "kong-revops",
		Format:     "table",
		Human:      true,
		QueryDir:   "/custom/path/to/queries",
	}
	if got != want {
		t.Fatalf("valid file: got %+v, want %+v", got, want)
	}
}

func TestLoadFromPartial(t *testing.T) {
	// All fields are optional -- a config that sets only one field should
	// leave the rest at their zero value, not error.
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`connection = "kong-revops"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("partial file: unexpected error: %v", err)
	}
	want := Config{Connection: "kong-revops"}
	if got != want {
		t.Fatalf("partial file: got %+v, want %+v", got, want)
	}
}
