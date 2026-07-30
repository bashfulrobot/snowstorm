# snowstorm

Snowflake data-access CLI. Connects via a named connection from
`~/.snowflake/connections.toml`, runs SQL, and returns structured JSON on
stdout. It's a data-access tool, not a reporting tool -- other tools
consume its JSON.

## Setup

Uses the same `~/.snowflake/connections.toml` the Snowflake connectors read, e.g.:

```toml
[my_connection]
account = "YOUR_ACCOUNT_LOCATOR"
user = "you@example.com"
authenticator = "externalbrowser"
role = "SOME_ROLE"
warehouse = "SOME_WAREHOUSE"
database = "SOME_DB"
schema = "SOME_SCHEMA"

# Optional: caches the SSO id token so externalbrowser doesn't reopen a
# browser on every run -- reused automatically as long as it's still valid.
# Auto-enabled on Windows/macOS; on Linux it defaults OFF and needs this
# explicit flag. No keyring/Secret Service daemon required on Linux -- the
# driver caches it in a plain file (0600, owned by you) under
# $SF_TEMPORARY_CREDENTIAL_CACHE_DIR, $XDG_CACHE_DIR/snowflake, or
# ~/.cache/snowflake by default.
client_store_temporary_credential = true

# Same idea, for authenticator = "username_password_mfa": caches the MFA
# token instead of the SSO id token. Same Linux-defaults-off caveat.
# client_request_mfa_token = true
```

Note the flat `[name]` table -- gosnowflake's own connections.toml loader wants this,
not the nested `[connections.name]` shape the Snowflake CLI/Python connector use.

`externalbrowser` opens a browser window for SSO -- run interactively, not headless/cron.
Run `snowstorm login` to establish or refresh that session explicitly (`snowstorm ping`
is for a quick connectivity check once you're already logged in, not for logging in).

These flags only control whether the token is *cached and reused*; how long the cached
token stays valid is entirely up to your Snowflake account's authentication/session
policies (server-side) -- snowstorm and gosnowflake have no client-side setting that
lengthens it. If you're still re-authenticating more often than expected with the flag
set, that's a policy question for your Snowflake account admin, not a snowstorm one.

## Usage

```sh
# log in / refresh an interactive session (browser popup for externalbrowser, if needed)
snowstorm login -c my_connection

# connectivity check (assumes you're already logged in)
snowstorm ping -c my_connection

# run SQL: inline, from a file, or piped via stdin
snowstorm query -c my_connection "SELECT * FROM MY_TABLE LIMIT 10"
snowstorm query -c my_connection --file query.sql
cat query.sql | snowstorm query -c my_connection

# --format table for a human-readable view (default is JSON)
snowstorm query -c my_connection --format table "SELECT 1"

# explore schema
snowstorm discover -c my_connection --database DB --schema SCHEMA
snowstorm discover -c my_connection --database DB --schema SCHEMA --table MY_TABLE --sample 20
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
snowstorm query -c my_connection --saved whoami
```

## Global flags

- `-c, --connection` -- named connection from connections.toml
- `--home` -- override the directory containing connections.toml
- `--timeout` -- timeout for the initial connection check
