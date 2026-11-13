# Usage

All commands take `-c, --connection`. Set `connection` in [config.toml](configuration.md) to omit it.

## Commands

| Command | Purpose |
| --- | --- |
| `login` | Establish or refresh an interactive session |
| `ping` | Verify connectivity, print session and account info |
| `query [SQL]` | Run SQL and print the result |
| `discover` | List tables, or a table's columns and sample rows |
| `queries list` | List [saved queries](saved-queries.md) |
| `logout` | Delete the cached SSO token |

## query

SQL comes from one of four places:

```sh
snowstorm query -c my_connection "SELECT * FROM MY_TABLE LIMIT 10"   # inline
snowstorm query -c my_connection --file query.sql                    # file
cat query.sql | snowstorm query -c my_connection                     # stdin
snowstorm query -c my_connection --saved whoami                      # saved query
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-f, --file` | | Read SQL from a file |
| `-s, --saved` | | Run a saved query by name |
| `--query-dir` | `~/.snowstorm/queries` | Saved query directory |
| `--format` | `table` | `table` or `json` |
| `--human` | `true` | Table only: abbreviate large numbers |
| `-o, --out` | `-` | Output path, `-` for stdout |
| `--skip-pick` | `false` | Never show the saved-query picker |
| `--skip-spinner` | `false` | Never show the progress spinner |

## discover

Explore schema. `--database` and `--schema` are required.

```sh
# list tables in a schema
snowstorm discover -c my_connection --database DB --schema SCHEMA

# list a table's columns, plus 20 sample rows
snowstorm discover -c my_connection --database DB --schema SCHEMA --table MY_TABLE --sample 20
```

| Flag | Meaning |
| --- | --- |
| `--database` | Database to inspect (required) |
| `--schema` | Schema to inspect (required) |
| `--table` | List this table's columns instead of tables |
| `--sample N` | With `--table`, also fetch N sample rows |
| `--format`, `--human`, `-o` | Same as `query` |

## ping

```sh
snowstorm ping -c my_connection
```

Accepts `--format`, `--human`, `-o`.

## Output

`query`, `ping`, `discover` and `queries list` default to `--format table --human`:
a readable table with comma-grouped, K/M/B/T-abbreviated numbers.

`--format json` returns the exact machine-parseable shape. `--human` and every
other display option leave it unchanged.

```sh
snowstorm query -c my_connection --format json "SELECT 1"
```

## Interactive niceties

On a real terminal (both stdin and stdout are TTYs):

- Picker: `snowstorm query` with no SQL, `--file` or `--saved` opens a picker
  over saved queries. It lists nothing until you type, then shows queries whose
  name starts with the typed text (case-insensitive), so other customers' names
  stay off screen. Enter runs the highlighted query. Ctrl-C or Esc exits with no output.
- Spinner: a stderr-only spinner shows while a `--format table` query runs.
- Colored errors: command errors get a colored `Error:` prefix.

None of this triggers for a non-interactive caller (piped or redirected stdin
or stdout). That check is hard and independent of any flag. Opt-outs for the
rare case: `--skip-pick`, `--skip-spinner`, `--no-color` (or `NO_COLOR`).

## Scripts and agents

Use `--format json` and the opt-outs for the stable path:

```sh
snowstorm query -c my_connection --saved whoami --format json --skip-pick --skip-spinner --no-color
```

## Global flags

| Flag | Meaning |
| --- | --- |
| `-c, --connection` | Named connection from `connections.toml` |
| `--home` | Directory containing `connections.toml` |
| `--timeout` | Timeout for the initial connection check (default 30s) |
| `--no-color` | Disable colored `Error:` prefix (also `NO_COLOR`) |
| `-v, --version` | Print the version and exit. Release builds show the tag, Nix builds show `devel-<revision>` |
