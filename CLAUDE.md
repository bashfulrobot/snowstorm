# snowstorm

Snowflake data-access CLI in Go. It runs SQL through a named connection in `~/.snowflake/connections.toml`.

## Build and test

- `just build` builds `./snowstorm`.
- `just test` runs `go test ./...`.
- `just check` runs vet, test and verify-driver, the same checks CI runs.
- `just` lists every recipe.

## Docs

`README.md` and `docs/`. The bundled Claude Code skill is documented in `docs/claude-skill.md`. `internal/gosnowflake` is a vendored driver, see `docs/development.md` before editing it.
