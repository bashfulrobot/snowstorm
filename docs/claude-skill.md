# Claude Code skill

`skills/snowstorm/SKILL.md` is a standalone Claude Code skill. It teaches
Claude to use the CLI: JSON output, saved queries, schema discovery, auth
errors and safe query habits.

## Install

Install the `snowstorm` binary first (see [Installation](installation.md)).
Then copy the skill into your personal skills directory:

```sh
mkdir -p ~/.claude/skills/snowstorm
curl -fsSL https://raw.githubusercontent.com/bashfulrobot/snowstorm/main/skills/snowstorm/SKILL.md \
  -o ~/.claude/skills/snowstorm/SKILL.md
```

Start a new Claude Code session. The skill loads when you ask to query
Snowflake or mention `snowstorm`. Re-run the `curl` to update.

For one project only, put the file at `<project>/.claude/skills/snowstorm/SKILL.md`.
Inside a clone of this repo, copy or symlink `skills/snowstorm` into `.claude/skills/`.

## What the skill does

- Runs `snowstorm query` with `--format json --skip-pick` so output is exact and never hangs on the picker.
- Resolves saved query names from `snowstorm queries list` instead of guessing.
- Explores schema with `snowstorm discover` before writing SQL.
- Confirms entity names with an `ILIKE` check before filtering on them.
- Reports error `390195` as an expired session and asks you to run `snowstorm login`.
- Avoids writes, unbounded queries and reading the token cache.

The skill never holds credentials. It calls the CLI, which uses your own
`~/.snowflake/connections.toml`.
