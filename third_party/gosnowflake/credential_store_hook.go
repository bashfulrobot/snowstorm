package gosnowflake

// snowstorm patch: the only change to the vendored gosnowflake v2.1.0 sources.
// Upstream chooses the credential storage backend by GOOS at package init and
// exposes no way to override it (macOS always uses the login keychain). This
// file adds an exported hook so a host program can supply its own store.

// CredentialStore is a key/value store for the driver's cached tokens (SSO ID
// token, MFA token, OAuth tokens). kind is the token type ("ID_TOKEN",
// "MFA_TOKEN", "OAUTH_ACCESS_TOKEN", "OAUTH_REFRESH_TOKEN"); key is an opaque
// hex digest the driver builds from host, user and kind; values are the token
// strings. The driver passes no lifetime information. Implementations must be
// safe for concurrent use and must not log values.
type CredentialStore interface {
	Get(kind, key string) string
	Set(kind, key, value string)
	Delete(kind, key string)
}

// SetCredentialStore replaces the driver's credential storage. Call it before
// opening any connection. A nil store is ignored.
func SetCredentialStore(store CredentialStore) {
	if store == nil {
		return
	}
	credentialsStorage = &hookedSecureStorageManager{store: store}
}

type hookedSecureStorageManager struct {
	store CredentialStore
}

func (m *hookedSecureStorageManager) setCredential(tokenSpec *secureTokenSpec, value string) {
	if value == "" {
		return
	}
	key, err := tokenSpec.buildKey()
	if err != nil {
		logger.Warnf("cannot build token spec: %v", err)
		return
	}
	m.store.Set(string(tokenSpec.tokenType), key, value)
}

func (m *hookedSecureStorageManager) getCredential(tokenSpec *secureTokenSpec) string {
	key, err := tokenSpec.buildKey()
	if err != nil {
		logger.Warnf("cannot build token spec: %v", err)
		return ""
	}
	return m.store.Get(string(tokenSpec.tokenType), key)
}

func (m *hookedSecureStorageManager) deleteCredential(tokenSpec *secureTokenSpec) {
	key, err := tokenSpec.buildKey()
	if err != nil {
		logger.Warnf("cannot build token spec: %v", err)
		return
	}
	m.store.Delete(string(tokenSpec.tokenType), key)
}
