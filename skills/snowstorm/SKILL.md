---
name: snowstorm
description: >-
  Query Snowflake through the local `snowstorm` CLI: run inline, file or saved
  SQL, explore schema with `snowstorm discover`, and read results as table or
  JSON. Use when the user asks to query Snowflake, pull numbers from a
  warehouse, list saved queries, explore a database or schema, or mentions
  `snowstorm`.
effort: low
---

# snowstorm

`snowstorm` is a Snowflake CLI. It connects through a named connection in
`~/.snowflake/connections.toml` and runs SQL. Its `--help` text is the ground
truth: run `snowstorm <command> --help` when anything below looks off.

## Preflight

1. `snowstorm --help` must work. If the command is missing, point the user to
   the install guide: https://github.com/bashfulrobot/snowstorm/blob/main/docs/installation.md
2. A connection must exist. The user passes `-c <name>`, or sets `connection`
   in `~/.snowstorm/config.toml`. Never invent a connection name. If none is
   known, ask the user which one.

## Run SQL

```bash
snowstorm query -c <conn> --format json --skip-pick --skip-spinner --no-color "SELECT ..."
snowstorm query -c <conn> --format json --skip-pick --file query.sql
snowstorm query -c <conn> --format json --skip-pick --saved <name>
```

- Always pass `--skip-pick`. The interactive picker only opens when stdin and
  stdout are both terminals, so this is insurance against a hang.
- Use `--format json` to parse or aggregate. JSON is exact and never abbreviated.
- Use the default `--format table` to show the user numbers in chat.
- Table mode abbreviates numbers (`5B`, `1.2M`) by default and the
  abbreviation rounds. For anything precise (billing, quotas, contract checks)
  use `--format json` or add `--human=false`.
- Only `table` and `json` exist. There is no `--format csv`. For CSV, pull JSON
  and convert it.
- Keep stderr out of the JSON. Redirect it (`2>"$TMPDIR/snowstorm.err"`) and
  read that file when the exit code is non-zero.

## Saved queries

Saved queries are `.sql` or `.toml` files in the query directory
(`~/.snowstorm/queries` by default).

```bash
snowstorm queries list --format json
```

Resolve names from that output. Do not guess a name and do not hardcode the
directory: `--query-dir`, `$SNOWSTORM_QUERY_DIR` and `config.toml` can all
override it.

To save a new query, write a file into the query directory. A leading `--`
comment in a `.sql` file becomes the description shown by `queries list`.

## Explore the schema

When no saved query fits, explore before writing SQL. Never guess table or
column names.

```bash
snowstorm discover -c <conn> --database DB --schema SCHEMA                      # tables
snowstorm discover -c <conn> --database DB --schema SCHEMA --table T --sample 20 # columns + rows
```

`--database` and `--schema` are required. Use a `LIMIT` on exploratory SQL.

## Verify identifiers that look alike

Names in a filter (accounts, customers, users) must match the stored string
exactly. A near-duplicate name returns another entity's rows or zero rows, with
no error. Before filtering on a name, confirm it:

```sql
SELECT DISTINCT NAME_COLUMN FROM DB.SCHEMA.TABLE WHERE NAME_COLUMN ILIKE '%partial%'
```

If more than one row comes back, show the candidates to the user and let them pick.

## Auth

Error `390195` (expired or invalid ID token) means the cached SSO session
expired. Tell the user to run `snowstorm login -c <conn>`. It opens a browser
for SSO, so an agent cannot complete it headless. Retrying the query does not
help. `snowstorm ping` checks connectivity for an already logged-in session and
does not log in.

## Do not

- Run `snowstorm logout` unless the user asks. It deletes the local token.
- Print, read or copy the token cache file or `connections.toml` contents into
  the conversation.
- Send writes (`INSERT`, `UPDATE`, `DELETE`, `DROP`, `CREATE`) unless the user
  asked for that specific change. Show the statement first.
- Run unbounded queries on large tables. Add `LIMIT` or a date filter.

## Reference

Full docs: https://github.com/bashfulrobot/snowstorm/tree/main/docs
