set shell := ["bash", "-euo", "pipefail", "-c"]

# list recipes
default:
    @just --list

# build ./snowstorm for this machine
build:
    go build -trimpath -ldflags "-s -w" -o snowstorm .

# run the unit tests
test:
    go test ./...

# vet the code
vet:
    go vet ./...

# lint (needs golangci-lint)
lint:
    golangci-lint run

# check the vendored driver against upstream (needs network or a warm module cache)
verify-driver:
    scripts/verify-vendored-gosnowflake.sh
    go test ./internal/gosnowflake/ -run 'SetCredentialStore|ClearIDToken'

# everything CI runs
check: vet test verify-driver

# cross-compile the four release targets into dist/ (no goreleaser needed)
build-all:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p dist
    for t in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
        os="${t%/*}"; arch="${t#*/}"
        echo "building $os/$arch"
        CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "-s -w" \
            -o "dist/snowstorm_${os}_${arch}" .
    done

# dry-run the release locally (needs goreleaser); builds archives in dist/, publishes nothing
release-snapshot:
    goreleaser release --snapshot --clean

# tag and push a release, e.g. `just release v0.2.0`; the release workflow does the rest
release version:
    #!/usr/bin/env bash
    set -euo pipefail
    [[ "{{version}}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "version must look like v1.2.3"; exit 1; }
    [[ "$(git rev-parse --abbrev-ref HEAD)" == "main" ]] || { echo "release from main"; exit 1; }
    git diff --quiet || { echo "working tree is dirty"; exit 1; }
    just check
    git tag -a "{{version}}" -m "{{version}}"
    git push origin "{{version}}"

# remove build output
clean:
    rm -rf dist snowstorm
