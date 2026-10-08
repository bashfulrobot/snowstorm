#!/usr/bin/env bash
# Verify internal/gosnowflake is upstream gosnowflake v2.1.0 minus the
# documented deletions, with the import path prefix rewritten (the one
# deterministic edit), plus exactly three added files. Fails on any other
# difference, and on any embedded minicore blob whose SHA-256 differs from the
# one recorded in SNOWSTORM_PATCH.md. Needs network or a warm module cache.
# Bash 3.2 compatible.
set -euo pipefail

version="v2.1.0"
mod="github.com/snowflakedb/gosnowflake/v2"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
vendored="$root/internal/gosnowflake"
note="$vendored/SNOWSTORM_PATCH.md"

# Run outside this module so the replace directive is not applied.
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
json="$(cd "$tmp" && go mod download -json "$mod@$version")"
upstream_src="$(printf '%s\n' "$json" | sed -n 's/.*"Dir": "\(.*\)",/\1/p')"
sum="$(printf '%s\n' "$json" | sed -n 's/.*"Sum": "\(.*\)",/\1/p')"
[ -d "$upstream_src" ] || { echo "cannot locate upstream $mod@$version" >&2; exit 1; }
# Apply the same deterministic import path rewrite to a copy of upstream.
upstream="$tmp/upstream"
cp -R "$upstream_src" "$upstream"
chmod -R u+w "$upstream"
find "$upstream" -name '*.go' -type f -exec sed -i.bak \
  's#github.com/snowflakedb/gosnowflake/v2#github.com/bashfulrobot/snowstorm/internal/gosnowflake#g' {} +
find "$upstream" -name '*.bak' -type f -delete
grep -q "$sum" "$note" || { echo "upstream dir hash $sum not recorded in SNOWSTORM_PATCH.md" >&2; exit 1; }

added="SNOWSTORM_PATCH.md credential_store_hook.go credential_store_hook_test.go"
fail=0

# Files only upstream must be documented deletions.
is_deleted_ok() {
  case "$1" in
    *_test.go|test_data/*|*/test_data/*|ci/*|cmd/*|arrowbatches/*|.github/*|.cursor/*|.windsurf/*) return 0 ;;
    .gitignore|.golangci.yml|.pre-commit-config.yaml|Jenkinsfile|parameters.json.*) return 0 ;;
    go.mod|go.sum|data1.txt.gz|codecov.yml|gosnowflake.mak|Makefile|CONTRIBUTING.md) return 0 ;;
  esac
  return 1
}

while IFS= read -r f; do
  if [ ! -e "$vendored/$f" ]; then
    is_deleted_ok "$f" || { echo "undocumented deletion: $f" >&2; fail=1; }
  elif [ -f "$upstream/$f" ] && ! cmp -s "$upstream/$f" "$vendored/$f"; then
    echo "modified: $f" >&2; fail=1
  fi
done < <(cd "$upstream" && find . -type f | sed 's|^\./||' | sort)

# Files only here must be exactly the three added ones.
while IFS= read -r f; do
  [ -e "$upstream/$f" ] && continue
  case " $added " in *" $f "*) ;; *) echo "unexpected added file: $f" >&2; fail=1 ;; esac
done < <(cd "$vendored" && find . -type f | sed 's|^\./||' | sort)
for f in $added; do
  [ -f "$vendored/$f" ] || { echo "missing added file: $f" >&2; fail=1; }
done

# Embedded blobs must match the recorded hashes.
for blob in "$vendored"/libsf_mini_core_*; do
  h="$(shasum -a 256 "$blob" | cut -d' ' -f1)"
  grep -q "$h  $(basename "$blob")" "$note" || { echo "blob hash not recorded: $(basename "$blob")" >&2; fail=1; }
done

if [ "$fail" -ne 0 ]; then echo "vendored gosnowflake verification FAILED" >&2; exit 1; fi
echo "vendored gosnowflake matches upstream $version (import path rewritten) plus the 3 documented additions"
