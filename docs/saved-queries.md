# Saved queries

Save a query by name instead of retyping it. Queries live in `--query-dir`
(default `~/.snowstorm/queries`, or `$SNOWSTORM_QUERY_DIR`) as plain `.sql` or structured `.toml`.

## SQL file

```sql
-- ~/.snowstorm/queries/whoami.sql
-- quick session identity check
SELECT CURRENT_USER() AS user, CURRENT_ROLE() AS role
```

## TOML file

```toml
# ~/.snowstorm/queries/warehouses.toml
name = "warehouses"
description = "list all warehouses"
sql = "SHOW WAREHOUSES"
```

## Run and list

```sh
snowstorm queries list
snowstorm query -c my_connection --saved whoami
```

`queries list` accepts `--query-dir`, `--format`, `--human` and `-o`.

On a terminal, `snowstorm query` with no arguments opens a prefix-match picker
over these queries. See [Usage](usage.md#interactive-niceties).

Examples live in the repo under `examples/queries`.
