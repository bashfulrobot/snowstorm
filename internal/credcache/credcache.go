// Package credcache is a file-backed store for gosnowflake's cached tokens
// (the SSO ID token for authenticator = "externalbrowser", MFA and OAuth
// tokens).
//
// Why it exists: on macOS gosnowflake keeps these in the login keychain via
// 99designs/keyring with an empty trusted-application list, and recreates the
// item whenever a login fails, so "Always Allow" never sticks and the user is
// prompted for the keychain password over and over. The driver has no switch
// for this, so snowstorm injects this store through the SetCredentialStore
// hook in third_party/gosnowflake. On Linux the driver already uses a file
// cache, so nothing is installed there.
//
// Tradeoff: the token sits in a plaintext JSON file (dir 0700, file 0600,
// owned by you), the same posture as gosnowflake's own Linux behaviour.
//
// The file format ({"tokens": {"<key>": "<token>"}}) matches the driver's
// Linux cache. Token values are never logged or put in error messages.
package credcache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/snowflakedb/gosnowflake/v2"
)

const (
	// EnvStore selects the backend on macOS: unset or "file" uses this
	// package, "keychain" restores gosnowflake's default login-keychain store.
	EnvStore = "SNOWSTORM_CREDENTIAL_STORE"

	// EnvDir overrides the directory holding the cache file.
	EnvDir = "SNOWSTORM_CREDENTIAL_CACHE_DIR"

	storeKeychain = "keychain"
	storeFile     = "file"

	dirName  = "snowstorm"
	fileName = "credential_cache_v1.json"
)

// Store is a file-backed gosnowflake.CredentialStore. Safe for concurrent use
// within a process; writes are atomic (temp file + rename) so concurrent
// snowstorm processes cannot leave a torn file (last writer wins).
type Store struct {
	dir string
	mu  sync.Mutex
}

var _ gosnowflake.CredentialStore = (*Store)(nil)

// New returns a Store that keeps its file in dir. Nothing touches disk until
// the first Set.
func New(dir string) *Store { return &Store{dir: dir} }

// DefaultDir returns the directory for the cache file: $SNOWSTORM_CREDENTIAL_CACHE_DIR
// if set, else <user cache dir>/snowstorm (~/Library/Caches/snowstorm on macOS).
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

// Install makes gosnowflake use a file store instead of the login keychain.
// It does nothing off macOS (Linux already uses a file cache in the driver;
// Windows keeps Credential Manager) or when $SNOWSTORM_CREDENTIAL_STORE is
// "keychain". If the cache directory cannot be determined it falls back to the
// driver's default rather than failing the command.
func Install() error {
	return install(runtime.GOOS, os.Getenv(EnvStore), gosnowflake.SetCredentialStore)
}

func install(goos, mode string, set func(gosnowflake.CredentialStore)) error {
	if goos != "darwin" {
		return nil
	}
	switch mode {
	case "", storeFile:
	case storeKeychain:
		return nil
	default:
		return fmt.Errorf("credcache: %s=%q: want %q or %q", EnvStore, mode, storeFile, storeKeychain)
	}
	dir, err := DefaultDir()
	if err != nil {
		return err
	}
	set(New(dir))
	return nil
}

func (s *Store) path() string { return filepath.Join(s.dir, fileName) }

// Get returns the token stored under key, or "" if there is none, the file is
// missing or unreadable, corrupted, or its permissions are looser than 0600.
func (s *Store) Get(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	tokens, _ := s.read()
	return tokens[key]
}

// Set stores value under key. Failures are swallowed (the driver treats the
// cache as best effort): the worst case is another browser login.
func (s *Store) Set(key, value string) {
	if key == "" || value == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tokens, _ := s.read()
	tokens[key] = value
	_ = s.write(tokens)
}

// Delete removes key. A missing key or file is not an error.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tokens, ok := s.read()
	if _, found := tokens[key]; !ok || !found {
		return
	}
	delete(tokens, key)
	_ = s.write(tokens)
}

type fileData struct {
	Tokens map[string]string `json:"tokens"`
}

// read loads the token map. It always returns a usable (non-nil) map; ok is
// false when the file was absent, unsafe or corrupt and the map is empty.
func (s *Store) read() (tokens map[string]string, ok bool) {
	tokens = map[string]string{}
	p := s.path()
	info, err := os.Stat(p)
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
	return fd.Tokens, true
}

func (s *Store) write(tokens map[string]string) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	// MkdirAll leaves an existing directory's mode alone; tighten it.
	if err := os.Chmod(s.dir, 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(fileData{Tokens: tokens})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, fileName+".tmp-*") // created 0600
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()
	if err = tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(raw)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return errors.New("credcache: write cache file failed")
	}
	if err = os.Rename(tmpName, s.path()); err != nil {
		return errors.New("credcache: replace cache file failed")
	}
	return nil
}
