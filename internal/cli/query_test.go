package cli

import "testing"

func TestResolveFormat(t *testing.T) {
	cases := []struct {
		name        string
		flagChanged bool
		flagValue   string
		configValue string
		want        string
	}{
		{"zero config, no flag: builtin default is now table", false, "table", "", "table"},
		{"explicit flag wins over config", true, "json", "table", "json"},
		{"config used when flag not passed", false, "table", "json", "json"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveFormat(tc.flagChanged, tc.flagValue, tc.configValue)
			if got != tc.want {
				t.Fatalf("resolveFormat(%v, %q, %q) = %q, want %q",
					tc.flagChanged, tc.flagValue, tc.configValue, got, tc.want)
			}
		})
	}
}

func TestResolveHuman(t *testing.T) {
	cases := []struct {
		name        string
		flagChanged bool
		flagValue   bool
		configValue bool
		want        bool
	}{
		{"zero config, no flag: builtin default is now true", false, true, false, true},
		{"explicit --human=true wins over config false", true, true, false, true},
		{"explicit --human=false wins over config true", true, false, true, false},
		{"config true used when flag not passed: still true", false, true, true, true},
		{"config false indistinguishable from unset, no flag: builtin default true wins", false, true, false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveHuman(tc.flagChanged, tc.flagValue, tc.configValue)
			if got != tc.want {
				t.Fatalf("resolveHuman(%v, %v, %v) = %v, want %v",
					tc.flagChanged, tc.flagValue, tc.configValue, got, tc.want)
			}
		})
	}
}

func TestResolveQueryDir(t *testing.T) {
	cases := []struct {
		name        string
		flagChanged bool
		flagValue   string
		envValue    string
		configValue string
		want        string
	}{
		{
			name:        "zero config, zero env, zero flag: empty (queryDir() applies its own default)",
			flagChanged: false, flagValue: "", envValue: "", configValue: "",
			want: "",
		},
		{
			name:        "explicit flag wins over everything",
			flagChanged: true, flagValue: "/flag/dir", envValue: "/env/dir", configValue: "/config/dir",
			want: "/flag/dir",
		},
		{
			name:        "env var wins over config when flag not passed",
			flagChanged: false, flagValue: "", envValue: "/env/dir", configValue: "/config/dir",
			want: "/env/dir",
		},
		{
			name:        "config used when flag not passed and no env var",
			flagChanged: false, flagValue: "", envValue: "", configValue: "/config/dir",
			want: "/config/dir",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveQueryDir(tc.flagChanged, tc.flagValue, tc.envValue, tc.configValue)
			if got != tc.want {
				t.Fatalf("resolveQueryDir(%v, %q, %q, %q) = %q, want %q",
					tc.flagChanged, tc.flagValue, tc.envValue, tc.configValue, got, tc.want)
			}
		})
	}
}
