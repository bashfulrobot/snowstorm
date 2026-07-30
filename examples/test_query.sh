#!/usr/bin/env bash
# Smoke-test the query command against a real predefined query file.
# Usage: examples/test_query.sh [connection-name]
set -euo pipefail

connection="${1:-kong-revops}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
query_file="$script_dir/queries/konnect_buy_vs_use_monthly.txt"

echo "==> snowstorm query --file $query_file (--format table)"
go run "$script_dir/.." query -c "$connection" --file "$query_file" --format table

echo
echo "==> same query, JSON output"
go run "$script_dir/.." query -c "$connection" --file "$query_file"
