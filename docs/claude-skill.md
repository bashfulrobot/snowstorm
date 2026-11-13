# Claude Code skill

The repo is a Claude Code plugin marketplace. It ships one plugin, `snowstorm`,
with one skill that teaches Claude to use the CLI: JSON output, saved queries,
schema discovery, auth errors and safe query habits.

## Install

Install the `snowstorm` binary first (see [Installation](installation.md)), then in Claude Code:

```
/plugin marketplace add bashfulrobot/snowstorm
/plugin install snowstorm@snowstorm
```

Claude loads the skill when you ask to query Snowflake or mention `snowstorm`.
Update with `/plugin marketplace update snowstorm`.

## What the skill does

- Runs `snowstorm query` with `--format json --skip-pick` so output is exact and never hangs on the picker.
- Resolves saved query names from `snowstorm queries list` instead of guessing.
- Explores schema with `snowstorm discover` before writing SQL.
- Confirms entity names with an `ILIKE` check before filtering on them.
- Reports error `390195` as an expired session and asks you to run `snowstorm login`.
- Avoids writes, unbounded queries and reading the token cache.

The skill never holds credentials. It calls the CLI, which uses your own
`~/.snowflake/connections.toml`.

## Layout

| Path | Contents |
| --- | --- |
| `.claude-plugin/marketplace.json` | Marketplace manifest |
| `plugins/snowstorm/.claude-plugin/plugin.json` | Plugin manifest |
| `plugins/snowstorm/skills/snowstorm/SKILL.md` | The skill |
