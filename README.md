# snowstorm

Snowflake data-access CLI. It connects through a named connection in `~/.snowflake/connections.toml` and runs SQL.

Output defaults to a readable table with abbreviated numbers and a prefix-match
picker for saved queries. `--format json` returns exact machine-parseable output
for scripts and agents.

## Quickstart

Install a [release binary](docs/installation.md#release-binary) (macOS and Linux,
amd64 and arm64), or with Go 1.26.5+:

```sh
go install github.com/bashfulrobot/snowstorm@latest
```

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

```
/plugin marketplace add bashfulrobot/snowstorm
/plugin install snowstorm@snowstorm
```

## Docs

- [Installation](docs/installation.md): release binary, go install, Nix, build from source
- [Configuration](docs/configuration.md): `connections.toml`, `config.toml`, env vars
- [Authentication](docs/authentication.md): login, token cache, logout, security tradeoffs
- [Usage](docs/usage.md): commands, flags, output formats, picker, scripting
- [Saved queries](docs/saved-queries.md): `.sql` and `.toml` queries
- [Claude Code skill](docs/claude-skill.md): install the plugin so Claude can drive snowstorm
- [Development](docs/development.md): build, test, vendored driver, releases

## License

See [LICENSE](LICENSE).
