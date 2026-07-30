package cli

import "testing"

func TestResolveConnection(t *testing.T) {
	cases := []struct {
		name        string
		flagChanged bool
		flagValue   string
		envValue    string
		configValue string
		want        string
	}{
		{
			name:        "zero config, zero env, zero flag: existing default (empty) behavior",
			flagChanged: false, flagValue: "", envValue: "", configValue: "",
			want: "",
		},
		{
			name:        "explicit flag wins over everything",
			flagChanged: true, flagValue: "flag-conn", envValue: "env-conn", configValue: "config-conn",
			want: "flag-conn",
		},
		{
			name:        "env var wins over config when flag not passed",
			flagChanged: false, flagValue: "", envValue: "env-conn", configValue: "config-conn",
			want: "env-conn",
		},
		{
			name:        "config used when flag not passed and no env var",
			flagChanged: false, flagValue: "", envValue: "", configValue: "config-conn",
			want: "config-conn",
		},
		{
			name:        "explicit empty flag value still counts as changed",
			flagChanged: true, flagValue: "", envValue: "env-conn", configValue: "config-conn",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveConnection(tc.flagChanged, tc.flagValue, tc.envValue, tc.configValue)
			if got != tc.want {
				t.Fatalf("resolveConnection(%v, %q, %q, %q) = %q, want %q",
					tc.flagChanged, tc.flagValue, tc.envValue, tc.configValue, got, tc.want)
			}
		})
	}
}
