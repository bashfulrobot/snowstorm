#!/usr/bin/env bash
# Smoke-test the query command against a predefined query file. Fill in
# the placeholders in queries/licensed_vs_consumed_monthly.txt first
# (YOUR_DB.YOUR_SCHEMA.YOUR_USAGE_TABLE, YOUR_ACCOUNT_NAME) -- this is a
# template, not a runnable-as-is query.
# Usage: examples/test_query.sh [connection-name]
set -euo pipefail

connection="${1:-my_connection}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
query_file="$script_dir/queries/licensed_vs_consumed_monthly.txt"

echo "==> snowstorm query --file $query_file (--format table)"
go run "$script_dir/.." query -c "$connection" --file "$query_file" --format table

echo
echo "==> same query, JSON output"
go run "$script_dir/.." query -c "$connection" --file "$query_file"
