# Development

## Build and test

[`just`](https://github.com/casey/just) wraps the common tasks. Run `just` to list them.

| Recipe | Does |
| --- | --- |
| `just build` | Build `./snowstorm` for this machine |
| `just test` | Run `go test ./...` |
| `just vet` | Run `go vet ./...` |
| `just lint` | Run `golangci-lint` |
| `just verify-driver` | Check the vendored driver against upstream |
| `just check` | `vet`, `test` and `verify-driver`, the same checks CI runs |
| `just build-all` | Cross-compile the four release targets into `dist/` |
| `just release-snapshot` | Dry-run GoReleaser locally, publishes nothing |
| `just release v0.2.0` | Run `check`, tag, push the tag |

Without `just`: `go build ./... && go test ./...`.

`nix develop` provides go, gopls, gotools, golangci-lint and gofumpt.

## CI

`.github/workflows/ci.yml` runs on every pull request and push to `main`: `go vet`,
`go test`, the vendored driver check, and a cross-build of the four release targets.

## Vendored driver

`internal/gosnowflake` must stay upstream v2.1.0 minus documented deletions,
with its import path prefix rewritten to this module, plus three added files.
See `internal/gosnowflake/SNOWSTORM_PATCH.md`.

The driver lives inside this module (not a `replace`d one) so
`go install github.com/bashfulrobot/snowstorm@latest` works.

Run these by hand before merging changes that touch the driver:

```sh
scripts/verify-vendored-gosnowflake.sh
go test ./internal/gosnowflake/ -run 'SetCredentialStore|ClearIDToken'
```

## Releases

Pushing a `vX.Y.Z` tag runs `.github/workflows/release.yml`, which builds
darwin and linux archives (amd64, arm64) with GoReleaser and attaches them to a
GitHub release with `checksums.txt`.

```sh
just release v0.2.0
```

`just release` refuses to run off `main` or with a dirty tree, runs `just check`,
then tags and pushes. Preview the archives first with `just release-snapshot`.

## Layout

| Path | Contents |
| --- | --- |
| `main.go` | Entry point |
| `internal/cli` | Cobra commands, output, picker, spinner |
| `internal/config` | `config.toml` loading |
| `internal/credcache` | macOS file token cache |
| `internal/gosnowflake` | Patched vendored driver |
| `internal/query`, `internal/snow` | Query execution and Snowflake access |
| `examples` | Sample saved queries |
