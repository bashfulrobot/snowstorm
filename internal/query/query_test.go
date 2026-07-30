package query

import (
	"reflect"
	"testing"
	"time"
)

func TestNormalize(t *testing.T) {
	ts := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name   string
		value  any
		dbType string
		want   any
	}{
		{"nil passthrough", nil, "TEXT", nil},
		{"number passthrough", int64(42), "FIXED", int64(42)},
		{"string passthrough", "hello", "TEXT", "hello"},
		{"timestamp formatted", ts, "TIMESTAMP_NTZ", "2026-07-29T12:00:00Z"},
		{"binary base64", []byte{0xDE, 0xAD, 0xBE, 0xEF}, "BINARY", "3q2+7w=="},
		{
			"variant object decoded",
			`{"a":1,"b":"two"}`,
			"VARIANT",
			map[string]any{"a": float64(1), "b": "two"},
		},
		{"array decoded", `[1,2,3]`, "ARRAY", []any{float64(1), float64(2), float64(3)}},
		{"variant invalid json falls back to string", "not json", "VARIANT", "not json"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalize(tc.value, tc.dbType)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("normalize(%#v, %q) = %#v, want %#v", tc.value, tc.dbType, got, tc.want)
			}
		})
	}
}
