package credcache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/snowflakedb/gosnowflake/v2"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return New(filepath.Join(t.TempDir(), "cache"))
}

func TestRoundTrip(t *testing.T) {
	s := newTestStore(t)
	s.Set("k1", "tok-one")
	s.Set("k2", "tok-two")
	if got := s.Get("k1"); got != "tok-one" {
		t.Fatalf("k1 = %q, want tok-one", got)
	}
	if got := s.Get("k2"); got != "tok-two" {
		t.Fatalf("k2 = %q, want tok-two", got)
	}
	s.Set("k1", "tok-new")
	if got := s.Get("k1"); got != "tok-new" {
		t.Fatalf("k1 after overwrite = %q", got)
	}
	// A fresh Store on the same dir sees the persisted data.
	if got := New(s.dir).Get("k2"); got != "tok-two" {
		t.Fatalf("persisted k2 = %q", got)
	}
	s.Delete("k1")
	if got := s.Get("k1"); got != "" {
		t.Fatalf("k1 after delete = %q", got)
	}
	if got := s.Get("k2"); got != "tok-two" {
		t.Fatalf("k2 lost by deleting k1: %q", got)
	}
}

func TestPermissions(t *testing.T) {
	s := newTestStore(t)
	s.Set("k", "v")
	di, err := os.Stat(s.dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := di.Mode().Perm(); got != 0o700 {
		t.Errorf("dir perm = %o, want 700", got)
	}
	fi, err := os.Stat(s.path())
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("file perm = %o, want 600", got)
	}
	entries, _ := os.ReadDir(s.dir)
	if len(entries) != 1 {
		t.Errorf("leftover files in cache dir: %d entries", len(entries))
	}
}

func TestTightensExistingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	New(dir).Set("k", "v")
	di, _ := os.Stat(dir)
	if got := di.Mode().Perm(); got != 0o700 {
		t.Errorf("dir perm = %o, want 700", got)
	}
}

func TestMissingFile(t *testing.T) {
	s := newTestStore(t)
	if got := s.Get("k"); got != "" {
		t.Fatalf("Get on missing file = %q", got)
	}
	s.Delete("k") // must not panic or create the file
	if _, err := os.Stat(s.path()); !os.IsNotExist(err) {
		t.Fatalf("Delete created the file: %v", err)
	}
}

func TestCorruptedFile(t *testing.T) {
	for name, content := range map[string]string{
		"garbage":    "not json at all",
		"truncated":  `{"tokens":{"k":"v`,
		"wrong type": `{"tokens":"nope"}`,
		"empty":      "",
	} {
		t.Run(name, func(t *testing.T) {
			s := newTestStore(t)
			if err := os.MkdirAll(s.dir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(s.path(), []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := s.Get("k"); got != "" {
				t.Fatalf("Get on corrupt file = %q", got)
			}
			s.Delete("k") // must not panic
			// Set recovers by replacing the corrupt file.
			s.Set("k", "v")
			if got := s.Get("k"); got != "v" {
				t.Fatalf("Get after recovery = %q", got)
			}
		})
	}
}

func TestLoosePermissionsFileIgnoredThenTightened(t *testing.T) {
	s := newTestStore(t)
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.path(), []byte(`{"tokens":{"k":"v"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(s.path(), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := s.Get("k"); got != "" {
		t.Fatalf("Get trusted a world-readable file: %q", got)
	}
	s.Set("k2", "v2")
	fi, _ := os.Stat(s.path())
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("file perm after Set = %o, want 600", got)
	}
}

func TestEmptyKeyOrValueIgnored(t *testing.T) {
	s := newTestStore(t)
	s.Set("", "v")
	s.Set("k", "")
	if _, err := os.Stat(s.path()); !os.IsNotExist(err) {
		t.Fatalf("file created for empty key/value: %v", err)
	}
}

func TestDefaultDir(t *testing.T) {
	t.Setenv(EnvDir, "/custom/dir")
	if got, err := DefaultDir(); err != nil || got != "/custom/dir" {
		t.Fatalf("override: got %q, %v", got, err)
	}
	t.Setenv(EnvDir, "")
	got, err := DefaultDir()
	if err != nil {
		t.Skipf("no user cache dir in this environment: %v", err)
	}
	if filepath.Base(got) != "snowstorm" {
		t.Errorf("default dir = %q, want .../snowstorm", got)
	}
}

func TestInstallSelection(t *testing.T) {
	t.Setenv(EnvDir, t.TempDir())
	called := false
	set := func(gosnowflake.CredentialStore) { called = true }

	cases := []struct {
		goos, mode string
		want       bool
		wantErr    bool
	}{
		{"darwin", "", true, false},
		{"darwin", "file", true, false},
		{"darwin", "keychain", false, false},
		{"darwin", "bogus", false, true},
		{"linux", "", false, false},
		{"windows", "", false, false},
	}
	for _, c := range cases {
		called = false
		err := install(c.goos, c.mode, set)
		if (err != nil) != c.wantErr || called != c.want {
			t.Errorf("install(%s,%q): called=%v err=%v, want called=%v err=%v", c.goos, c.mode, called, err, c.want, c.wantErr)
		}
	}
}
