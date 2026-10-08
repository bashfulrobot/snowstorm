# snowstorm

Snowflake data-access CLI. Connects via a named connection from
`~/.snowflake/connections.toml` and runs SQL. Output is tuned for a human
at a terminal by default (a readable table, abbreviated numbers, an
fzf-style picker for saved queries) -- pass `--format json` for the old
byte-for-byte machine-parseable output any script or agent should use.

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
# explicit flag. See "Where the cached token lives" below.
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

### Where the cached token lives

- **macOS (default): a file cache.** gosnowflake would use the login keychain,
  but it creates that item with an empty trusted-app list and recreates it
  after every expired login, so "Always Allow" never sticks and you get a
  keychain password prompt over and over. snowstorm ships a patched copy of the
  driver (`third_party/gosnowflake`) and stores the token in a file instead.
  **This is a plaintext token file and it is on by default.**
- **Opt out (back to the keychain):** `credential_store = "keychain"` in
  `~/.snowstorm/config.toml`, or `SNOWSTORM_CREDENTIAL_STORE=keychain` in the
  environment. The env var wins over config.toml; `file` is the other valid
  value (and the default); anything else is an error. `connections.toml` is not
  touched.
- **Linux:** unchanged. gosnowflake keeps the token in its own plain file
  (0600, owned by you) under `$SF_TEMPORARY_CREDENTIAL_CACHE_DIR`,
  `$XDG_CACHE_DIR/snowflake`, or `~/.cache/snowflake`.
- **Windows:** unchanged (Credential Manager).

File cache details:

- Location: `~/Library/Caches/snowstorm/credential_cache_v1.json`, in the user
  cache dir where tools keep regenerable data. That is under your home
  directory; whether a backup or sync tool skips `~/Library/Caches` is up to
  that tool, so exclude it yourself if it matters. Directory 0700, file 0600,
  atomic writes with an exclusive (`O_EXCL`) temp file; orphaned temp files are
  cleaned up. Symlinks (file, temp file, directory) are refused, and a
  directory or file owned by another user is never used or chmod-ed.
- `SNOWSTORM_CREDENTIAL_CACHE_DIR=/abs/path` moves the cache: it must be
  absolute, and snowstorm always appends its own `snowstorm` subdirectory
  (`/abs/path/snowstorm/`) so it only ever manages a directory it owns.
- Only the short-lived SSO ID token is stored. MFA and OAuth tokens the driver
  offers are not persisted. The driver passes no token lifetime, so each entry
  records its write time and anything older than 24 hours is ignored (the
  Snowflake server-side expiry still applies on top).
- A failed login (for example a rejected or expired token, error 390195)
  makes the driver delete the token through the store.
- `snowstorm logout` deletes the locally cached token (no Snowflake call; the
  server-side session is not revoked). In file mode it removes the cache file
  and also asks the login keychain to drop a stale pre-upgrade item. In keychain
  mode, or on Linux/Windows, it says where the token lives and asks the
  driver's own storage to delete the token for the resolved connection (the
  driver does not report whether one existed; a keychain delete may prompt).
- Security tradeoff: it is a plaintext token. The 0600 mode stops other users,
  not other processes running as you; the keychain would prompt before handing
  the token to such a process, this does not. A token left in the keychain by
  earlier runs is not read or removed in file mode; delete it in Keychain
  Access if you want it gone.

### Checking the vendored driver

`third_party/gosnowflake` must stay upstream v2.1.0 minus documented deletions
plus three added files (see `third_party/gosnowflake/SNOWSTORM_PATCH.md`). There
is no CI workflow in this repo, so run these by hand before merging changes
that touch it:

```sh
scripts/verify-vendored-gosnowflake.sh
(cd third_party/gosnowflake && go test -run 'SetCredentialStore|ClearIDToken' .)
```

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

# --format json for the old machine-parseable output (default is now table)
snowstorm query -c my_connection --format json "SELECT 1"

# explore schema
snowstorm discover -c my_connection --database DB --schema SCHEMA
snowstorm discover -c my_connection --database DB --schema SCHEMA --table MY_TABLE --sample 20
```

### Human-first defaults, agent escape hatches

`query`, `ping`, `discover`, and `queries list` all default to `--format
table --human` (comma-grouped and K/M/B/T-abbreviated numbers, no flags
needed). `--format json` always returns the old byte-for-byte exact shape,
unaffected by `--human` or anything else here -- that's the path a script
or agent should use.

Running `snowstorm query` with no SQL, `--file`, or `--saved`, in a real
terminal (both stdin and stdout), opens an fzf-style fuzzy picker over your
saved queries instead of reading stdin -- type to filter, Enter runs the
highlighted query, Ctrl-C/Esc exits cleanly with no output. A stderr-only
spinner shows while a `--format table` query runs. Command errors get a
colorized `Error:` prefix on a real terminal.

None of this ever triggers for a non-interactive caller (piped/redirected
stdin or stdout, no TTY) -- that's a hard check independent of any flag.
On top of it, explicit opt-outs exist for the rare case you want them
anyway: `--skip-pick`, `--skip-spinner`, `--no-color` (or the standard
`NO_COLOR` env var).

```sh
# full agent/script path: unchanged since before these defaults existed
snowstorm query -c my_connection --saved whoami --format json --skip-pick --skip-spinner --no-color
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

### Tool config

`~/.snowstorm/config.toml` sets defaults for `--connection`, `--format`,
`--human`, and `--query-dir` so they don't need to be passed every time.
All fields optional; explicit flags always win over it, which wins over
env vars where those exist (`$SNOWSTORM_QUERY_DIR`), which wins over the
builtin default.

```toml
# ~/.snowstorm/config.toml
connection = "kong-revops"
format     = "table"
human      = true
query_dir  = "/custom/path/to/queries"
# credential_store = "keychain"   # macOS: opt out of the default file token cache (see above)
```

## Global flags

- `-c, --connection` -- named connection from connections.toml
- `--home` -- override the directory containing connections.toml
- `--timeout` -- timeout for the initial connection check
- `--no-color` -- disable the colorized `Error:` prefix on command errors (also respects `NO_COLOR`)
