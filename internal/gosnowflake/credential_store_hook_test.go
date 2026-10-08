package gosnowflake

import "testing"

type recordingStore struct {
	calls []string
	val   string
}

func (r *recordingStore) Get(kind, key string) string {
	r.calls = append(r.calls, "get:"+kind)
	return r.val
}
func (r *recordingStore) Set(kind, key, value string) { r.calls = append(r.calls, "set:"+kind) }
func (r *recordingStore) Delete(kind, key string)     { r.calls = append(r.calls, "del:"+kind) }

// Mirrors what authenticateWithConfig does: the same calls go through the
// package-level credentialsStorage, including the delete on a failed login
// (which is what an expired/rejected ID token, code 390195, ends up in).
func TestSetCredentialStoreRoutesDriverCalls(t *testing.T) {
	orig := credentialsStorage
	defer func() { credentialsStorage = orig }()

	r := &recordingStore{val: "x"}
	SetCredentialStore(r)

	spec := newIDTokenSpec("host.example", "user")
	credentialsStorage.setCredential(spec, "tok")
	credentialsStorage.setCredential(spec, "") // empty values are never forwarded
	if got := credentialsStorage.getCredential(spec); got != "x" {
		t.Fatalf("get = %q", got)
	}
	credentialsStorage.deleteCredential(spec)
	credentialsStorage.setCredential(newMfaTokenSpec("host.example", "user"), "m")

	want := []string{"set:ID_TOKEN", "get:ID_TOKEN", "del:ID_TOKEN", "set:MFA_TOKEN"}
	if len(r.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", r.calls, want)
	}
	for i := range want {
		if r.calls[i] != want[i] {
			t.Fatalf("calls = %v, want %v", r.calls, want)
		}
	}

	credentialsStorage.setCredential(&secureTokenSpec{tokenType: idToken}, "tok") // no host/user: key build fails
	if len(r.calls) != len(want) {
		t.Fatalf("store called despite key build failure: %v", r.calls)
	}
}

func TestClearIDTokenUsesDefaultStorageEvenWhenHooked(t *testing.T) {
	orig, origDefault := credentialsStorage, defaultCredentialsStorage
	defer func() { credentialsStorage, defaultCredentialsStorage = orig, origDefault }()

	def := &recordingStore{}
	defaultCredentialsStorage = &hookedSecureStorageManager{store: def}
	hooked := &recordingStore{}
	SetCredentialStore(hooked)

	ClearIDToken("host.example", "user")
	ClearIDToken("", "user") // no-op
	ClearIDToken("host.example", "")
	if len(def.calls) != 1 || def.calls[0] != "del:ID_TOKEN" {
		t.Fatalf("default store calls = %v", def.calls)
	}
	if len(hooked.calls) != 0 {
		t.Fatalf("hooked store touched by ClearIDToken: %v", hooked.calls)
	}
}
