# snowstorm patch

This is gosnowflake v2.1.0 vendored as a regular package of this module (a
`replace` directive would break `go install github.com/bashfulrobot/snowstorm@latest`).
Compared with upstream: test files, test data, ci/, cmd/, the arrowbatches
submodule, go.mod/go.sum and upstream dotfiles/CI config are removed; the
import path prefix `github.com/snowflakedb/gosnowflake/v2` is rewritten to
`github.com/bashfulrobot/snowstorm/internal/gosnowflake` in every `.go` file
(the only edit to upstream content); and three files are added:

- `credential_store_hook.go`: exports `SetCredentialStore` (swap the credential
  storage backend) and `ClearIDToken` (delete the cached ID token through the
  OS-default storage, e.g. the macOS keychain).
- `credential_store_hook_test.go`: the only test kept. It runs as part of
  `go test ./...`, or on its own with
  `go test ./internal/gosnowflake/ -run 'SetCredentialStore|ClearIDToken'`
- `SNOWSTORM_PATCH.md`: this note.

snowstorm uses the hook by default on macOS to keep the SSO ID token in a file
instead of the login keychain (see `internal/credcache`).

## Verification

`scripts/verify-vendored-gosnowflake.sh` re-downloads upstream, applies the same import path rewrite, diffs it against
this directory and fails on any difference other than the documented deletions
and the three added files, and checks the blob hashes below.

Upstream: `github.com/snowflakedb/gosnowflake/v2@v2.1.0`
Module dir hash (`go mod download -json` Sum): `h1:rfjs6NAMnbLKCBYlOarqQX/UKgQVrXi43TZNHCP5/jw=`

Embedded minicore blobs (`go:embed`), SHA-256:

```
cf1ace398487fda1d56242ebb7ab4c8e01aff6369950e35d9bcce6b1209a3dc1  libsf_mini_core_darwin_amd64.dylib
da320a4cf4ac60295f9ab1c0bf417d798ac96dbf5855ea98e8ca2753c3ae49eb  libsf_mini_core_darwin_arm64.dylib
2242aaa0949a2afdba1ea825c71cfe3e867ee1644e9b3c77568cd23b7d3f2fa5  libsf_mini_core_linux_amd64_glibc.so
5e5f5f3d54ca7552c052dcc26479d46a0f9cb06f388c19874fd1a5960dbff035  libsf_mini_core_linux_amd64_musl.so
6dbd623c6eb1e7149dd0bbd47bece80f7c1ffd3e3c8f8bc101a1e25ec958a93a  libsf_mini_core_linux_arm64_glibc.so
38e10753f46f430c961a80ac16c3b24fd1867eeee2e48e5023d5bd7da9a30f52  libsf_mini_core_linux_arm64_musl.so
cbecbd9403c9e2cc693e597e89e478fbd92bd785d11cee1c40f3d3246510374b  libsf_mini_core_windows_amd64.dll
b055cf70c86b04ce2ac361fffdd526ea6611ae467750be7247cae0208d9550ca  libsf_mini_core_windows_arm64.dll
```

## Upgrading

Copy the new release over this directory (same exclusions, then the same
import path rewrite), keep the three
added files, update the version, hashes and the script's `version`, and check
that `credentialsStorage` and the `secureStorageManager` interface still have
the same shape.
