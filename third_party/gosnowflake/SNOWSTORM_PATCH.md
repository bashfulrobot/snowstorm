# snowstorm patch

This is gosnowflake v2.1.0 with test files, test data, ci/, cmd/ and the
arrowbatches submodule removed, plus one added file:
`credential_store_hook.go` (exports `SetCredentialStore`). No upstream file is
modified. snowstorm uses the hook on macOS to keep the SSO ID token in a file
instead of the login keychain (see `internal/credcache`).

To upgrade: copy the new release over this directory (same exclusions), keep
`credential_store_hook.go`, and check that `credentialsStorage` and the
`secureStorageManager` interface still have the same shape.
