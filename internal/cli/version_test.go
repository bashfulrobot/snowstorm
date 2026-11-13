package cli

import (
	"runtime/debug"
	"testing"
)

func TestVersionFrom(t *testing.T) {
	withMain := func(v string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Version: v}}
	}
	withVCS := func(rev, modified string) *debug.BuildInfo {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "(devel)"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: rev},
				{Key: "vcs.modified", Value: modified},
			},
		}
	}

	tests := []struct {
		name     string
		override string
		info     *debug.BuildInfo
		ok       bool
		want     string
	}{
		{"override wins", "v9.9.9", withMain("v0.1.1"), true, "v9.9.9"},
		{"module version", "", withMain("v0.1.1"), true, "v0.1.1"},
		{"devel module, clean vcs", "", withVCS("0123456789abcdef", "false"), true, "devel-0123456"},
		{"devel module, dirty vcs", "", withVCS("0123456789abcdef", "true"), true, "devel-0123456-dirty"},
		{"short revision kept whole", "", withVCS("abc12", "false"), true, "devel-abc12"},
		{"devel with no vcs", "", withMain("(devel)"), true, "devel"},
		{"empty module version", "", withMain(""), true, "devel"},
		{"no build info", "", nil, false, "devel"},
		{"ok without info", "", nil, true, "devel"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionFrom(tt.override, tt.info, tt.ok); got != tt.want {
				t.Errorf("versionFrom() = %q, want %q", got, tt.want)
			}
		})
	}
}
