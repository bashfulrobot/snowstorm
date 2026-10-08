// Package credcache is an opt-in, file-backed store for gosnowflake's cached
// SSO ID token (authenticator = "externalbrowser").
//
// Why it exists: on macOS gosnowflake keeps the ID token in the login keychain
// via 99designs/keyring with an empty trusted-application list, and recreates
// the item whenever a login fails, so "Always Allow" never sticks and the user
// is prompted for the keychain password over and over. The driver has no
// switch for this, so snowstorm injects this store through the
// SetCredentialStore hook in third_party/gosnowflake. It is OFF by default:
// the keychain stays the default and the file store is enabled with
// credential_store = "file" in ~/.snowstorm/config.toml or
// SNOWSTORM_CREDENTIAL_STORE=file (env wins). On Linux the driver already uses
// its own file cache, so nothing is installed there.
//
// Security posture: the token sits in a plaintext JSON file (dir 0700, file
// 0600). That stops other users, not other processes running as you. Only the
// ID token is ever persisted (MFA and OAuth tokens that the driver offers are
// dropped), entries older than MaxAge are ignored, and symlinks are refused
// for the file, the temp file and the directory. Token values are never
// logged or put in error messages.
package credcache

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/snowflakedb/gosnowflake/v2"
)

const (
	// EnvStore selects the backend: "file" or "keychain" (the default). It
	// wins over config.toml's credential_store.
	EnvStore = "SNOWSTORM_CREDENTIAL_STORE"

	// EnvDir overrides the directory holding the cache file.
	EnvDir = "SNOWSTORM_CREDENTIAL_CACHE_DIR"

	// ModeFile and ModeKeychain are the valid credential_store values.
	ModeFile     = "file"
	ModeKeychain = "keychain"

	// KindIDToken is the only token type that is persisted.
	KindIDToken = "ID_TOKEN"

	// MaxAge bounds how long a written token is trusted. The driver passes
	// no lifetime, so this is a ceiling on top of the server-side expiry.
	MaxAge = 24 * time.Hour

	dirName  = "snowstorm"
	fileName = "credential_cache_v1.json"
)

// now is replaced in tests.
var now = time.Now

// Resolve picks the credential store mode: the env value if set, else the
// config.toml value, else the default (keychain). Unknown values are errors.
func Resolve(envValue, configValue string) (string, error) {
	mode, src := configValue, "credential_store"
	if envValue != "" {
		mode, src = envValue, EnvStore
	}
	switch mode {
	case "":
		return ModeKeychain, nil
	case ModeFile, ModeKeychain:
		return mode, nil
	}
	return "", fmt.Errorf("credcache: %s = %q: want %q or %q", src, mode, ModeFile, ModeKeychain)
}

// Store is a file-backed gosnowflake.CredentialStore. Safe for concurrent use
// within a process; writes are atomic (exclusive temp file + rename) so
// concurrent snowstorm processes cannot leave a torn file (last writer wins).
type Store struct {
	dir string
	mu  sync.Mutex
}

var _ gosnowflake.CredentialStore = (*Store)(nil)

// New returns a Store that keeps its file in dir. Nothing touches disk until
// the first Set.
func New(dir string) *Store { return &Store{dir: dir} }

// DefaultDir returns the directory for the cache file: $SNOWSTORM_CREDENTIAL_CACHE_DIR
// if set, else <user cache dir>/snowstorm (~/Library/Caches/snowstorm on
// macOS). The default is outside any home-directory sync or backup tree.
func DefaultDir() (string, error) {
	if d := os.Getenv(EnvDir); d != "" {
		return d, nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("credcache: locate user cache dir: %w", err)
	}
	return filepath.Join(base, dirName), nil
}

// Install applies the already-resolved mode (see Resolve). Only ModeFile on
// macOS replaces the driver's storage; everywhere else the driver default
// stays (login keychain on macOS, its own file cache on Linux, Credential
// Manager on Windows).
func Install(mode string) error {
	return install(runtime.GOOS, mode, gosnowflake.SetCredentialStore)
}

func install(goos, mode string, set func(gosnowflake.CredentialStore)) error {
	if _, err := Resolve(mode, ""); err != nil {
		return err
	}
	if goos != "darwin" || mode != ModeFile {
		return nil
	}
	dir, err := DefaultDir()
	if err != nil {
		return err
	}
	set(New(dir))
	return nil
}

// Clear deletes the cache file in the default directory. A missing file is
// not an error; a symlink at that path is removed as a link, never followed.
func Clear() error {
	dir, err := DefaultDir()
	if err != nil {
		return err
	}
	return New(dir).Clear()
}

// Clear deletes this store's cache file.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("credcache: remove cache file failed")
	}
	return nil
}

func (s *Store) path() string { return filepath.Join(s.dir, fileName) }

// Get returns the stored ID token for key, or "" if there is none, it is
// older than MaxAge, the file is missing, unreadable, corrupt, a symlink, or
// has permissions looser than 0600, or kind is not the ID token.
func (s *Store) Get(kind, key string) string {
	if kind != KindIDToken {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tokens, _ := s.read()
	return tokens[key].Value
}

// Set stores an ID token. Other kinds are silently not persisted. Failures
// are swallowed (the driver treats the cache as best effort): the worst case
// is another browser login.
func (s *Store) Set(kind, key, value string) {
	if kind != KindIDToken || key == "" || value == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tokens, _ := s.read()
	tokens[key] = entry{Value: value, WrittenAt: now().Unix()}
	_ = s.write(tokens)
}

// Delete removes the stored ID token for key. If the cache file is unsafe or
// corrupt it is removed outright, since it cannot be trusted anyway.
func (s *Store) Delete(kind, key string) {
	if kind != KindIDToken {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tokens, ok := s.read()
	if !ok {
		if _, err := os.Lstat(s.path()); err == nil {
			_ = os.Remove(s.path())
		}
		return
	}
	if _, found := tokens[key]; !found {
		return
	}
	delete(tokens, key)
	_ = s.write(tokens)
}

type entry struct {
	Value     string `json:"v"`
	WrittenAt int64  `json:"at"` // unix seconds
}

type fileData struct {
	Tokens map[string]entry `json:"tokens"`
}

// read loads the live (unexpired) entries. It always returns a usable map; ok
// is false when the file was absent, unsafe or corrupt.
func (s *Store) read() (tokens map[string]entry, ok bool) {
	tokens = map[string]entry{}
	p := s.path()
	if di, err := os.Lstat(s.dir); err != nil || !di.IsDir() {
		return tokens, false
	}
	info, err := os.Lstat(p) // Lstat: a symlink is never regular
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return tokens, false
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return tokens, false
	}
	var fd fileData
	if err := json.Unmarshal(raw, &fd); err != nil || fd.Tokens == nil {
		return tokens, false
	}
	cutoff := now().Add(-MaxAge).Unix()
	for k, e := range fd.Tokens {
		if e.Value != "" && e.WrittenAt > cutoff && e.WrittenAt <= now().Add(time.Minute).Unix() {
			tokens[k] = e
		}
	}
	return tokens, true
}

func (s *Store) write(tokens map[string]entry) error {
	// Refuse a symlinked directory or file; never follow either.
	if di, err := os.Lstat(s.dir); err == nil && di.Mode()&os.ModeSymlink != 0 {
		return errors.New("credcache: cache dir is a symlink")
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	di, err := os.Lstat(s.dir)
	if err != nil || !di.IsDir() {
		return errors.New("credcache: cache dir is not a directory")
	}
	// MkdirAll leaves an existing directory's mode alone; tighten it.
	if err := os.Chmod(s.dir, 0o700); err != nil {
		return err
	}
	if fi, err := os.Lstat(s.path()); err == nil && !fi.Mode().IsRegular() {
		return errors.New("credcache: cache file is not a regular file")
	}
	raw, err := json.Marshal(fileData{Tokens: tokens})
	if err != nil {
		return err
	}

	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return err
	}
	tmpName := filepath.Join(s.dir, fileName+".tmp-"+hex.EncodeToString(suffix[:]))
	tmp, err := os.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("credcache: create temp file failed")
	}
	_, err = tmp.Write(raw)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmpName, s.path())
	}
	if err != nil {
		_ = os.Remove(tmpName)
		return errors.New("credcache: write cache file failed")
	}
	return nil
}
