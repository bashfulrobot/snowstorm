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
		{"zero config, no flag: builtin default", false, "json", "", "json"},
		{"explicit flag wins over config", true, "table", "json", "table"},
		{"config used when flag not passed", false, "json", "table", "table"},
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
		{"zero config, no flag: builtin default false", false, false, false, false},
		{"explicit --human=true wins over config false", true, true, false, true},
		{"explicit --human=false wins over config true", true, false, true, false},
		{"config true used when flag not passed", false, false, true, true},
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
