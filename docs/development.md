# Development

## Build and test

```sh
go build ./...
go test ./...
```

`nix develop` provides go, gopls, gotools, golangci-lint and gofumpt.

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
git tag v0.1.0
git push origin v0.1.0
```

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
