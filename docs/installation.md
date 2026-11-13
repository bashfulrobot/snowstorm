# Installation

snowstorm is a single Go binary. **Choose one** of the four options below.
Each option is an alternative to the others, so do not run more than one.

| Option | Best for | Needs |
| --- | --- | --- |
| 1. Release binary | Most users | `tar`, `curl` or a browser |
| 2. `go install` | Go developers | Go 1.26.9+ |
| 3. Nix | Nix users | Nix with flakes |
| 4. Build from source | Contributors | Go 1.26.9+, a clone |

## Option 1: Release binary

Download the archive for your OS and CPU from the
[Releases page](https://github.com/bashfulrobot/snowstorm/releases), then:

```sh
tar -xzf snowstorm_<version>_<os>_<arch>.tar.gz
sudo mv snowstorm /usr/local/bin/
```

Archives exist for `darwin` and `linux`, each on `amd64` and `arm64`. A
`checksums.txt` file sits next to them. Verify with `shasum -a 256 -c checksums.txt --ignore-missing`.

On macOS, a downloaded binary may be quarantined. Clear it with
`xattr -d com.apple.quarantine snowstorm`.

**OR**

## Option 2: go install

Needs Go 1.26.9 or newer (see `go.mod`). Older Go versions have standard library
vulnerabilities fixed in 1.26.9, so `go.mod` requires it.

```sh
go install github.com/bashfulrobot/snowstorm@latest
```

The binary lands in `$(go env GOBIN)`, or `$(go env GOPATH)/bin` when `GOBIN`
is unset. Put that directory on `PATH`.

**OR**

## Option 3: Nix

The repo ships a flake with a `snowstorm` package and a dev shell.

```sh
# run without installing
nix run github:bashfulrobot/snowstorm -- --help

# install into your profile
nix profile install github:bashfulrobot/snowstorm
```

To use it from another flake, add `github:bashfulrobot/snowstorm` as an input
and reference `packages.<system>.default`.

**OR**

## Option 4: Build from source

```sh
git clone https://github.com/bashfulrobot/snowstorm
cd snowstorm
go build -o snowstorm .
```

## Verify

After any option:

```sh
snowstorm --help
```

Next: [Configuration](configuration.md).
