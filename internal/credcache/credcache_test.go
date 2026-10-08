package credcache

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/snowflakedb/gosnowflake/v2"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return New(filepath.Join(t.TempDir(), "cache"))
}

func setNow(t *testing.T, tm time.Time) {
	t.Helper()
	orig := now
	now = func() time.Time { return tm }
	t.Cleanup(func() { now = orig })
}

func TestRoundTrip(t *testing.T) {
	s := newTestStore(t)
	s.Set(KindIDToken, "k1", "tok-one")
	s.Set(KindIDToken, "k2", "tok-two")
	if got := s.Get(KindIDToken, "k1"); got != "tok-one" {
		t.Fatalf("k1 = %q", got)
	}
	s.Set(KindIDToken, "k1", "tok-new")
	if got := s.Get(KindIDToken, "k1"); got != "tok-new" {
		t.Fatalf("k1 after overwrite = %q", got)
	}
	if got := New(s.dir).Get(KindIDToken, "k2"); got != "tok-two" {
		t.Fatalf("persisted k2 = %q", got)
	}
	s.Delete(KindIDToken, "k1")
	if got := s.Get(KindIDToken, "k1"); got != "" {
		t.Fatalf("k1 after delete = %q", got)
	}
	if got := s.Get(KindIDToken, "k2"); got != "tok-two" {
		t.Fatalf("k2 lost by deleting k1: %q", got)
	}
}

func TestOnlyIDTokenPersisted(t *testing.T) {
	s := newTestStore(t)
	for _, kind := range []string{"MFA_TOKEN", "OAUTH_ACCESS_TOKEN", "OAUTH_REFRESH_TOKEN", ""} {
		s.Set(kind, "k", "secret-"+kind)
		if got := s.Get(kind, "k"); got != "" {
			t.Errorf("kind %q readable: %q", kind, got)
		}
	}
	if _, err := os.Stat(s.path()); !os.IsNotExist(err) {
		t.Fatalf("file created for non-ID tokens: %v", err)
	}
	s.Set(KindIDToken, "id", "idtok")
	raw, _ := os.ReadFile(s.path())
	if strings.Contains(string(raw), "secret-") {
		t.Fatal("non-ID token reached disk")
	}
	// A non-ID delete must not remove the ID token under the same key.
	s.Delete("MFA_TOKEN", "id")
	if got := s.Get(KindIDToken, "id"); got != "idtok" {
		t.Fatalf("ID token removed by MFA delete: %q", got)
	}
}

func TestPermissions(t *testing.T) {
	s := newTestStore(t)
	s.Set(KindIDToken, "k", "v")
	di, _ := os.Stat(s.dir)
	if got := di.Mode().Perm(); got != 0o700 {
		t.Errorf("dir perm = %o, want 700", got)
	}
	fi, _ := os.Stat(s.path())
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("file perm = %o, want 600", got)
	}
	if entries, _ := os.ReadDir(s.dir); len(entries) != 1 {
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
	New(dir).Set(KindIDToken, "k", "v")
	di, _ := os.Stat(dir)
	if got := di.Mode().Perm(); got != 0o700 {
		t.Errorf("dir perm = %o, want 700", got)
	}
}

func TestMissingFile(t *testing.T) {
	s := newTestStore(t)
	if got := s.Get(KindIDToken, "k"); got != "" {
		t.Fatalf("Get on missing file = %q", got)
	}
	s.Delete(KindIDToken, "k")
	if _, err := os.Stat(s.path()); !os.IsNotExist(err) {
		t.Fatalf("Delete created the file: %v", err)
	}
}

func TestCorruptedFile(t *testing.T) {
	for name, content := range map[string]string{
		"garbage":    "not json at all",
		"truncated":  `{"tokens":{"k":{"v":"v`,
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
			if got := s.Get(KindIDToken, "k"); got != "" {
				t.Fatalf("Get on corrupt file = %q", got)
			}
			s.Set(KindIDToken, "k", "v") // recovers by replacing the file
			if got := s.Get(KindIDToken, "k"); got != "v" {
				t.Fatalf("Get after recovery = %q", got)
			}
		})
	}
}

func TestDeleteRemovesCorruptFile(t *testing.T) {
	s := newTestStore(t)
	_ = os.MkdirAll(s.dir, 0o700)
	_ = os.WriteFile(s.path(), []byte("junk"), 0o600)
	s.Delete(KindIDToken, "k")
	if _, err := os.Lstat(s.path()); !os.IsNotExist(err) {
		t.Fatalf("corrupt file survived Delete: %v", err)
	}
}

func TestLoosePermissionsFileIgnoredThenReplaced(t *testing.T) {
	s := newTestStore(t)
	_ = os.MkdirAll(s.dir, 0o700)
	data := `{"tokens":{"k":{"v":"v","at":` + strconv.FormatInt(time.Now().Unix(), 10) + `}}}`
	if err := os.WriteFile(s.path(), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(s.path(), 0o644)
	if got := s.Get(KindIDToken, "k"); got != "" {
		t.Fatalf("Get trusted a world-readable file: %q", got)
	}
	s.Set(KindIDToken, "k2", "v2")
	fi, _ := os.Stat(s.path())
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("file perm after Set = %o, want 600", got)
	}
}

func TestExpiry(t *testing.T) {
	base := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	s := newTestStore(t)
	setNow(t, base)
	s.Set(KindIDToken, "k", "v")

	setNow(t, base.Add(MaxAge-time.Minute))
	if got := s.Get(KindIDToken, "k"); got != "v" {
		t.Fatalf("fresh token not returned: %q", got)
	}
	setNow(t, base.Add(MaxAge+time.Minute))
	if got := s.Get(KindIDToken, "k"); got != "" {
		t.Fatalf("expired token returned: %q", got)
	}
	// Writing another entry prunes the expired one from disk.
	s.Set(KindIDToken, "k2", "v2")
	raw, _ := os.ReadFile(s.path())
	if strings.Contains(string(raw), `"k":`) {
		t.Fatal("expired entry not pruned on write")
	}
	// An entry stamped in the future is not trusted either.
	setNow(t, base.Add(-time.Hour))
	if got := s.Get(KindIDToken, "k2"); got != "" {
		t.Fatalf("future-dated token returned: %q", got)
	}
}

func TestRefusesSymlinks(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		s := newTestStore(t)
		_ = os.MkdirAll(s.dir, 0o700)
		target := filepath.Join(t.TempDir(), "target")
		_ = os.WriteFile(target, []byte("untouched"), 0o600)
		if err := os.Symlink(target, s.path()); err != nil {
			t.Skip("symlinks unsupported:", err)
		}
		s.Set(KindIDToken, "k", "v")
		if b, _ := os.ReadFile(target); string(b) != "untouched" {
			t.Fatal("wrote through a symlinked cache file")
		}
		if got := s.Get(KindIDToken, "k"); got != "" {
			t.Fatalf("read through symlink: %q", got)
		}
		s.Delete(KindIDToken, "k")
		if b, _ := os.ReadFile(target); string(b) != "untouched" {
			t.Fatal("Delete modified the symlink target")
		}
		if _, err := os.Stat(target); err != nil {
			t.Fatal("symlink target removed")
		}
	})
	t.Run("dir", func(t *testing.T) {
		real := t.TempDir()
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(real, link); err != nil {
			t.Skip("symlinks unsupported:", err)
		}
		s := New(link)
		s.Set(KindIDToken, "k", "v")
		if entries, _ := os.ReadDir(real); len(entries) != 0 {
			t.Fatal("wrote through a symlinked cache dir")
		}
		if got := s.Get(KindIDToken, "k"); got != "" {
			t.Fatalf("read through symlinked dir: %q", got)
		}
	})
	t.Run("temp file", func(t *testing.T) {
		// O_EXCL means a pre-planted symlink at the temp name is never followed.
		dir := t.TempDir()
		victim := filepath.Join(t.TempDir(), "victim")
		_ = os.WriteFile(victim, []byte("untouched"), 0o600)
		planted := filepath.Join(dir, "planted")
		if err := os.Symlink(victim, planted); err != nil {
			t.Skip("symlinks unsupported:", err)
		}
		f, err := os.OpenFile(planted, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			f.Close()
			t.Fatal("O_EXCL followed/ignored an existing symlink")
		}
		// And the store's own writes leave no temp files behind.
		s := New(dir)
		s.Set(KindIDToken, "k", "v")
		for _, e := range mustReadDir(t, dir) {
			if strings.Contains(e, ".tmp-") {
				t.Fatalf("temp file left behind: %s", e)
			}
		}
	})
}

func mustReadDir(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

func TestClear(t *testing.T) {
	s := newTestStore(t)
	if err := s.Clear(); err != nil {
		t.Fatalf("Clear on missing file: %v", err)
	}
	s.Set(KindIDToken, "k", "v")
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if got := s.Get(KindIDToken, "k"); got != "" {
		t.Fatalf("token survived Clear: %q", got)
	}
}

func TestEmptyKeyOrValueIgnored(t *testing.T) {
	s := newTestStore(t)
	s.Set(KindIDToken, "", "v")
	s.Set(KindIDToken, "k", "")
	if _, err := os.Stat(s.path()); !os.IsNotExist(err) {
		t.Fatalf("file created for empty key/value: %v", err)
	}
}

func TestDefaultDir(t *testing.T) {
	t.Setenv(EnvDir, "/custom/dir")
	if got, err := DefaultDir(); err != nil || got != "/custom/dir/snowstorm" {
		t.Fatalf("override: got %q, %v (want the snowstorm leaf appended)", got, err)
	}
	for _, rel := range []string{"relative/dir", ".", "~/x"} {
		t.Setenv(EnvDir, rel)
		if got, err := DefaultDir(); err == nil {
			t.Fatalf("relative override %q accepted: %q", rel, got)
		}
	}
	t.Setenv(EnvDir, "")
	got, err := DefaultDir()
	if err != nil {
		t.Skipf("no user cache dir: %v", err)
	}
	if filepath.Base(got) != "snowstorm" {
		t.Errorf("default dir = %q, want .../snowstorm", got)
	}
}

func TestResolve(t *testing.T) {
	cases := []struct {
		env, cfg, want string
		wantErr        bool
	}{
		{"", "", ModeFile, false}, // default is the file cache
		{"", "file", ModeFile, false},
		{"", "keychain", ModeKeychain, false},
		{"file", "", ModeFile, false},
		{"keychain", "file", ModeKeychain, false}, // env wins over config
		{"file", "keychain", ModeFile, false},
		{"", "keychain", ModeKeychain, false},
		{"bogus", "", "", true},
		{"", "bogus", "", true},
		{"file", "bogus", ModeFile, false}, // env wins, config not consulted
	}
	for _, c := range cases {
		got, err := Resolve(c.env, c.cfg)
		if got != c.want || (err != nil) != c.wantErr {
			t.Errorf("Resolve(%q,%q) = %q, %v; want %q err=%v", c.env, c.cfg, got, err, c.want, c.wantErr)
		}
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
		{"darwin", "", true, false}, // default on macOS is the file cache
		{"darwin", ModeFile, true, false},
		{"darwin", ModeKeychain, false, false}, // opt-out
		{"darwin", "bogus", false, true},
		{"linux", "", false, false}, // driver's own file cache
		{"linux", ModeFile, false, false},
		{"linux", "bogus", false, true},
		{"windows", "", false, false},
		{"windows", ModeFile, false, false},
	}
	for _, c := range cases {
		called = false
		err := install(c.goos, c.mode, set)
		if (err != nil) != c.wantErr || called != c.want {
			t.Errorf("install(%s,%q): called=%v err=%v", c.goos, c.mode, called, err)
		}
	}
}

func TestRefusesForeignOwnedDir(t *testing.T) {
	orig := ownedByUs
	defer func() { ownedByUs = orig }()

	dir := filepath.Join(t.TempDir(), "cache")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ownedByUs = func(os.FileInfo) bool { return false }
	s := New(dir)
	s.Set(KindIDToken, "k", "v")
	s.Delete(KindIDToken, "k")
	if err := s.Clear(); err == nil {
		t.Error("Clear accepted a foreign-owned dir")
	}
	di, _ := os.Stat(dir)
	if di.Mode().Perm() != 0o755 {
		t.Errorf("foreign-owned dir was chmod-ed to %o", di.Mode().Perm())
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("wrote into a foreign-owned dir: %d entries", len(entries))
	}
}

func TestIgnoresForeignOwnedFile(t *testing.T) {
	orig := ownedByUs
	defer func() { ownedByUs = orig }()

	s := newTestStore(t)
	s.Set(KindIDToken, "k", "v")
	ownedByUs = func(fi os.FileInfo) bool { return fi.IsDir() } // dir ours, file not
	if got := s.Get(KindIDToken, "k"); got != "" {
		t.Fatalf("read a foreign-owned cache file: %q", got)
	}
}

func TestClearRefusesSymlinkedDir(t *testing.T) {
	real := t.TempDir()
	victim := filepath.Join(real, fileName)
	if err := os.WriteFile(victim, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks unsupported:", err)
	}
	s := New(link)
	if err := s.Clear(); err == nil {
		t.Error("Clear followed a symlinked dir")
	}
	s.Delete(KindIDToken, "k")
	if b, err := os.ReadFile(victim); err != nil || string(b) != "keep" {
		t.Fatal("file behind a symlinked dir was touched")
	}
}

func TestOrphanedTempFilesCleaned(t *testing.T) {
	s := newTestStore(t)
	s.Set(KindIDToken, "k", "v")
	old := filepath.Join(s.dir, fileName+".tmp-old")
	fresh := filepath.Join(s.dir, fileName+".tmp-fresh")
	linkTmp := filepath.Join(s.dir, fileName+".tmp-link")
	unrelated := filepath.Join(s.dir, "other.tmp-x")
	for _, p := range []string{old, fresh, unrelated} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	haveLink := os.Symlink(unrelated, linkTmp) == nil

	// write(): removes old orphans, keeps recent ones (could be in flight).
	s.Set(KindIDToken, "k2", "v2")
	if _, err := os.Lstat(old); !os.IsNotExist(err) {
		t.Error("old orphan survived write")
	}
	if _, err := os.Lstat(fresh); err != nil {
		t.Error("in-flight temp file was removed by write")
	}
	if _, err := os.Lstat(unrelated); err != nil {
		t.Error("unrelated file removed")
	}
	if haveLink {
		if _, err := os.Lstat(linkTmp); err != nil {
			t.Error("symlink temp entry was removed or followed by write")
		}
	}

	// Clear(): removes all regular temp files and the cache file, never the
	// symlink's target.
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{fresh, s.path()} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Errorf("%s survived Clear", p)
		}
	}
	if _, err := os.Lstat(unrelated); err != nil {
		t.Error("Clear removed an unrelated file or a symlink target")
	}
}
