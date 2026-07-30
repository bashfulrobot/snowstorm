# snowstorm

Snowflake data-access CLI. Connects via a named connection from
`~/.snowflake/connections.toml`, runs SQL, and returns structured JSON on
stdout. It's a data-access tool, not a reporting tool -- other tools
consume its JSON.

## Setup

Uses the same `~/.snowflake/connections.toml` the Snowflake connectors read, e.g.:

```toml
[kong-revops]
account = "CGLFPQY-HVA68606"
user = "you@example.com"
authenticator = "externalbrowser"
role = "SOME_ROLE"
warehouse = "SOME_WAREHOUSE"
database = "SOME_DB"
schema = "SOME_SCHEMA"
```

Note the flat `[name]` table -- gosnowflake's own connections.toml loader wants this,
not the nested `[connections.name]` shape the Snowflake CLI/Python connector use.

`externalbrowser` opens a browser window for SSO -- run interactively, not headless/cron.

## Usage

```sh
# connectivity check
snowstorm ping -c kong-revops

# run SQL: inline, from a file, or piped via stdin
snowstorm query -c kong-revops "SELECT * FROM MY_TABLE LIMIT 10"
snowstorm query -c kong-revops --file query.sql
cat query.sql | snowstorm query -c kong-revops

# --format table for a human-readable view (default is JSON)
snowstorm query -c kong-revops --format table "SELECT 1"

# explore schema
snowstorm discover -c kong-revops --database DB --schema SCHEMA
snowstorm discover -c kong-revops --database DB --schema SCHEMA --table MY_TABLE --sample 20
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
snowstorm query -c kong-revops --saved whoami
```

## Global flags

- `-c, --connection` -- named connection from connections.toml
- `--home` -- override the directory containing connections.toml
- `--timeout` -- timeout for the initial connection check
