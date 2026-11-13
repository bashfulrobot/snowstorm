# Configuration

snowstorm reads two files. Only `connections.toml` is required.

| File | Purpose |
| --- | --- |
| `~/.snowflake/connections.toml` | Snowflake connection profiles (required) |
| `~/.snowstorm/config.toml` | snowstorm defaults (optional) |

## connections.toml

snowstorm uses the same `~/.snowflake/connections.toml` the Snowflake connectors read.

```toml
[my_connection]
account = "YOUR_ACCOUNT_LOCATOR"
user = "you@example.com"
authenticator = "externalbrowser"
role = "SOME_ROLE"
warehouse = "SOME_WAREHOUSE"
database = "SOME_DB"
schema = "SOME_SCHEMA"

# Optional: cache the SSO id token so externalbrowser does not reopen a
# browser on every run. Auto-enabled on Windows and macOS. Linux defaults
# OFF and needs this flag.
client_store_temporary_credential = true

# Optional: same idea for authenticator = "username_password_mfa".
# client_request_mfa_token = true
```

Use a flat `[name]` table. gosnowflake's loader rejects the nested
`[connections.name]` shape the Snowflake CLI and Python connector use.

`--home` points snowstorm at a different directory containing `connections.toml`.

## config.toml

`~/.snowstorm/config.toml` sets defaults so flags need not be repeated. All fields are optional.

```toml
connection = "my_connection"
format     = "table"          # or "json"
human      = true             # abbreviate large numbers in table output
query_dir  = "/custom/path/to/queries"
# credential_store = "keychain"   # macOS only, see authentication.md
```

Precedence, highest first:

1. Explicit flag
2. Environment variable (where one exists, for example `SNOWSTORM_QUERY_DIR`)
3. `config.toml`
4. Built-in default

## Environment variables

| Variable | Effect |
| --- | --- |
| `SNOWSTORM_QUERY_DIR` | Saved query directory |
| `SNOWSTORM_CREDENTIAL_STORE` | `file` (default) or `keychain`, macOS only |
| `SNOWSTORM_CREDENTIAL_CACHE_DIR` | Absolute path for the token cache base directory |
| `NO_COLOR` | Disable the colored `Error:` prefix |

Next: [Authentication](authentication.md), [Usage](usage.md).
