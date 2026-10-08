package cli

import (
	"errors"
	"strings"
	"testing"
)

func joined(l []string) string { return strings.Join(l, "\n") }

func TestLogoutReportFileMode(t *testing.T) {
	out := joined(logoutReport("darwin", "file", "/c/snowstorm", nil, "h", "u", nil))
	for _, want := range []string{"removed snowstorm's file-cached token from /c/snowstorm", "stale pre-file-cache", "not revoked"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestLogoutReportNeverClaimsFileRemovalOutsideFileMode(t *testing.T) {
	for _, c := range []struct{ goos, mode, where string }{
		{"darwin", "keychain", "login keychain"},
		{"linux", "file", "~/.cache/snowflake"},
		{"windows", "file", "Windows Credential Manager"},
	} {
		out := joined(logoutReport(c.goos, c.mode, "", nil, "h", "u", nil))
		if strings.Contains(out, "removed snowstorm's") {
			t.Errorf("%s/%s claims file removal:\n%s", c.goos, c.mode, out)
		}
		if !strings.Contains(out, c.where) || !strings.Contains(out, "file cache was not touched") {
			t.Errorf("%s/%s does not say where the token lives:\n%s", c.goos, c.mode, out)
		}
	}
}

func TestLogoutReportErrors(t *testing.T) {
	out := joined(logoutReport("darwin", "keychain", "", nil, "", "", errors.New("no connection")))
	if !strings.Contains(out, "nothing was removed from the login keychain") {
		t.Errorf("unresolved identity not reported honestly:\n%s", out)
	}
	out = joined(logoutReport("darwin", "file", "/c", errors.New("boom"), "h", "u", nil))
	if strings.Contains(out, "removed snowstorm's") || !strings.Contains(out, "could not remove") {
		t.Errorf("file error reported as success:\n%s", out)
	}
}
