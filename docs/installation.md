# Installation

snowstorm is a single Go binary. Pick one method.

## Release binary

Download the archive for your OS and CPU from the
[Releases page](https://github.com/bashfulrobot/snowstorm/releases), then:

```sh
tar -xzf snowstorm_<version>_<os>_<arch>.tar.gz
sudo mv snowstorm /usr/local/bin/
snowstorm --help
```

Archives exist for `darwin` and `linux`, each on `amd64` and `arm64`. A
`checksums.txt` file sits next to them. Verify with `shasum -a 256 -c checksums.txt --ignore-missing`.

On macOS, a downloaded binary may be quarantined. Clear it with
`xattr -d com.apple.quarantine snowstorm`.

## go install

Needs Go 1.26.5 or newer (see `go.mod`).

```sh
go install github.com/bashfulrobot/snowstorm@latest
```

The binary lands in `$(go env GOBIN)`, or `$(go env GOPATH)/bin` when `GOBIN`
is unset. Put that directory on `PATH`.

## Nix

The repo ships a flake with a `snowstorm` package and a dev shell.

```sh
# run without installing
nix run github:bashfulrobot/snowstorm -- --help

# install into your profile
nix profile install github:bashfulrobot/snowstorm
```

To use it from another flake, add `github:bashfulrobot/snowstorm` as an input
and reference `packages.<system>.default`.

## Build from source

```sh
git clone https://github.com/bashfulrobot/snowstorm
cd snowstorm
go build -o snowstorm .
```

## Verify

```sh
snowstorm --help
```

Next: [Configuration](configuration.md).
