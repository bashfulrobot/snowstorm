# snowstorm patch

This is gosnowflake v2.1.0 with test files, test data, ci/, cmd/ and the
arrowbatches submodule removed, plus two added files:
`credential_store_hook.go` (exports `SetCredentialStore`). No upstream file is
modified. snowstorm uses the hook on macOS to keep the SSO ID token in a file
instead of the login keychain by default on macOS (see `internal/credcache`).
`credential_store_hook_test.go` is the only test kept; it is outside the root
module, so run it with `cd third_party/gosnowflake && go test -run SetCredentialStore .`.

To upgrade: copy the new release over this directory (same exclusions), keep
`credential_store_hook.go`, and check that `credentialsStorage` and the
`secureStorageManager` interface still have the same shape.
