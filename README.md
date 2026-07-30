# snowstorm

Snowflake data-access CLI. Connects via a named connection from
`~/.snowflake/connections.toml`, runs SQL, and returns structured JSON on
stdout. It's a data-access tool, not a reporting tool -- other tools
consume its JSON.

## Setup

Uses the same `~/.snowflake/connections.toml` the Snowflake connectors read, e.g.:

```toml
[connections.my_example_connection]
account = "CGLFPQY-HVA68606"
user = "you@example.com"
authenticator = "externalbrowser"
role = "SOME_ROLE"
warehouse = "SOME_WAREHOUSE"
database = "SOME_DB"
schema = "SOME_SCHEMA"
```

`externalbrowser` opens a browser window for SSO -- run interactively, not headless/cron.

## Usage

```sh
# connectivity check
snowstorm ping -c my_example_connection

# run SQL: inline, from a file, or piped via stdin
snowstorm query -c my_example_connection "SELECT * FROM MY_TABLE LIMIT 10"
snowstorm query -c my_example_connection --file query.sql
cat query.sql | snowstorm query -c my_example_connection

# --format table for a human-readable view (default is JSON)
snowstorm query -c my_example_connection --format table "SELECT 1"

# explore schema
snowstorm discover -c my_example_connection --database DB --schema SCHEMA
snowstorm discover -c my_example_connection --database DB --schema SCHEMA --table MY_TABLE --sample 20
```

### Predefined queries

Save a query by name instead of retyping it. Queries live in
`--query-dir` (default `~/.snowstorm/queries`, or `$SNOWSTORM_QUERY_DIR`)
as either plain `.sql` or structured `.toml`:

```sql
-- ~/.snowstorm/queries/whoami.sql
-- quick session identity check
SELECT CURRENT_USER() AS user, CURRENT_ROLE() AS role
```

```toml
# ~/.snowstorm/queries/warehouses.toml
name = "warehouses"
description = "list all warehouses"
sql = "SHOW WAREHOUSES"
```

```sh
snowstorm queries list
snowstorm query -c my_example_connection --saved whoami
```

## Global flags

- `-c, --connection` -- named connection from connections.toml
- `--home` -- override the directory containing connections.toml
- `--timeout` -- timeout for the initial connection check
