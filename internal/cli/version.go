package cli

import "runtime/debug"

// Version is the release version. Release and Nix builds set it with
//
//	-ldflags "-X github.com/bashfulrobot/snowstorm/internal/cli.Version=v1.2.3"
//
// When it is empty, buildVersion falls back to what the Go toolchain embedded:
// the module version for `go install module@vX.Y.Z`, or the VCS revision for a
// build from a checkout.
var Version string

// buildVersion returns the version string shown by `snowstorm --version`.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	return versionFrom(Version, info, ok)
}

// versionFrom picks the best version in order: an explicit override, the main
// module version, the VCS revision, then "devel". It takes its inputs as
// arguments so tests can cover every branch.
func versionFrom(override string, info *debug.BuildInfo, ok bool) string {
	if override != "" {
		return override
	}
	if !ok || info == nil {
		return "devel"
	}
	// `go install module@vX.Y.Z` and release builds from a tag record the
	// version here. A local build records "(devel)" instead.
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var rev string
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if rev == "" {
		return "devel"
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if modified {
		return "devel-" + rev + "-dirty"
	}
	return "devel-" + rev
}
