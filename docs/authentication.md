# Authentication

`externalbrowser` opens a browser window for SSO. Run it interactively, not headless or from cron.

```sh
snowstorm login -c my_connection    # establish or refresh the session
snowstorm ping -c my_connection     # connectivity check once logged in
snowstorm logout -c my_connection   # delete the locally cached token
```

`ping` does not log in. Use `login` first.

## Token caching

`client_store_temporary_credential = true` (and `client_request_mfa_token = true`
for MFA) only controls whether the token is cached and reused. How long it stays
valid depends on your Snowflake account's server-side authentication and session
policies. snowstorm and gosnowflake have no client setting that lengthens it. If
you re-authenticate more often than expected, ask your Snowflake admin.

## Where the cached token lives

- **macOS (default): file cache.** gosnowflake would use the login keychain, but
  it creates the item with an empty trusted-app list and recreates it after each
  expired login. "Always Allow" never sticks and the keychain password prompt
  repeats. snowstorm ships a patched driver (`internal/gosnowflake`) that stores
  the token in a file instead. **This is a plaintext token file and it is on by default.**
- **Opt out (back to the keychain):** `credential_store = "keychain"` in
  `~/.snowstorm/config.toml`, or `SNOWSTORM_CREDENTIAL_STORE=keychain`. The env
  var wins. Valid values: `file`, `keychain`. Anything else is an error.
  `connections.toml` is untouched.
- **Linux:** unchanged. gosnowflake keeps the token in a plain file (0600, owned
  by you) under `$SF_TEMPORARY_CREDENTIAL_CACHE_DIR`, `$XDG_CACHE_DIR/snowflake`,
  or `~/.cache/snowflake`.
- **Windows:** unchanged (Credential Manager).

## File cache details (macOS)

- Location: `~/Library/Caches/snowstorm/credential_cache_v1.json`. Whether a
  backup or sync tool skips `~/Library/Caches` is up to that tool. Exclude it
  yourself if it matters.
- Directory mode 0700, file mode 0600. Writes are atomic with an exclusive
  (`O_EXCL`) temp file, and orphaned temp files are cleaned up.
- Symlinks (file, temp file, directory) are refused. A directory or file owned
  by another user is never used or chmod-ed.
- `SNOWSTORM_CREDENTIAL_CACHE_DIR=/abs/path` moves the cache. The path must be
  absolute. snowstorm appends its own `snowstorm` subdirectory
  (`/abs/path/snowstorm/`) so it only manages a directory it owns.
- Only the short-lived SSO ID token is stored. MFA and OAuth tokens are not persisted.
- Each entry records its write time. Entries older than 24 hours are ignored,
  and the Snowflake server-side expiry still applies.
- A failed login (for example error 390195, rejected or expired token) makes the
  driver delete the token.

## logout

`snowstorm logout` deletes the local token. It makes no Snowflake call, so the
server-side session is not revoked.

- File mode: removes the cache file and asks the login keychain to drop a stale
  pre-upgrade item.
- Keychain mode, Linux, Windows: reports where the token lives and asks the
  driver's storage to delete the token for the resolved connection. The driver
  does not report whether one existed. A keychain delete may prompt.

## Security tradeoff

The file cache is a plaintext token. Mode 0600 stops other users, not other
processes running as you. The keychain would prompt before handing the token to
such a process. The file does not. A token left in the keychain by earlier runs
is not read or removed in file mode. Delete it in Keychain Access if you want it gone.
