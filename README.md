# snowstorm

Snowflake data-access CLI. It connects through a named connection in `~/.snowflake/connections.toml` and runs SQL.

Output defaults to a readable table with abbreviated numbers and a prefix-match
picker for saved queries. `--format json` returns exact machine-parseable output
for scripts and agents.

## Quickstart

Install snowstorm for macOS or Linux. See [Installation](docs/installation.md)
for the options: release binary, `go install`, Nix, or build from source.

Add a connection to `~/.snowflake/connections.toml`:

```toml
[my_connection]
account = "YOUR_ACCOUNT_LOCATOR"
user = "you@example.com"
authenticator = "externalbrowser"
role = "SOME_ROLE"
warehouse = "SOME_WAREHOUSE"
```

Log in, check, query:

```sh
snowstorm login -c my_connection
snowstorm ping -c my_connection
snowstorm query -c my_connection "SELECT CURRENT_USER()"
snowstorm query -c my_connection --format json "SELECT 1"   # for scripts
```

Optional: let Claude Code drive snowstorm with the bundled skill.

```sh
mkdir -p ~/.claude/skills/snowstorm
curl -fsSL https://raw.githubusercontent.com/bashfulrobot/snowstorm/main/skills/snowstorm/SKILL.md \
  -o ~/.claude/skills/snowstorm/SKILL.md
```

## Docs

- [Installation](docs/installation.md): release binary, go install, Nix, build from source
- [Configuration](docs/configuration.md): `connections.toml`, `config.toml`, env vars
- [Authentication](docs/authentication.md): login, token cache, logout, security tradeoffs
- [Usage](docs/usage.md): commands, flags, output formats, picker, scripting
- [Saved queries](docs/saved-queries.md): `.sql` and `.toml` queries
- [Claude Code skill](docs/claude-skill.md): install the skill so Claude can drive snowstorm
- [Development](docs/development.md): build, test, vendored driver, releases

## License

See [LICENSE](LICENSE).
