package cli

import (
	"strings"
	"testing"

	"github.com/bashfulrobot/snowstorm/internal/query"
)

func TestLoginConfirmation(t *testing.T) {
	res := &query.Result{
		Columns: []string{"LOGIN_USER", "LOGIN_ACCOUNT", "LOGIN_ROLE"},
		Rows: []map[string]any{
			{"LOGIN_USER": "ALICE", "LOGIN_ACCOUNT": "MYACCT", "LOGIN_ROLE": "SYSADMIN"},
		},
		RowCount: 1,
	}

	got, err := loginConfirmation(res)
	if err != nil {
		t.Fatalf("loginConfirmation() unexpected error: %v", err)
	}
	want := "Logged in as ALICE on account MYACCT (role SYSADMIN)."
	if got != want {
		t.Fatalf("loginConfirmation() = %q, want %q", got, want)
	}
}

func TestLoginConfirmationErrorsOnUnexpectedRowCount(t *testing.T) {
	cases := []struct {
		name string
		res  *query.Result
	}{
		{"nil result", nil},
		{"zero rows", &query.Result{Rows: []map[string]any{}}},
		{
			"more than one row",
			&query.Result{Rows: []map[string]any{
				{"LOGIN_USER": "ALICE"},
				{"LOGIN_USER": "BOB"},
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loginConfirmation(tc.res)
			if err == nil {
				t.Fatal("loginConfirmation() expected an error, got nil")
			}
			if !strings.Contains(err.Error(), "expected exactly one row") {
				t.Fatalf("loginConfirmation() error = %v, want it to mention the row-count expectation", err)
			}
		})
	}
}

func TestLoginConfirmationHandlesMissingFields(t *testing.T) {
	res := &query.Result{
		Rows: []map[string]any{{}},
	}

	got, err := loginConfirmation(res)
	if err != nil {
		t.Fatalf("loginConfirmation() unexpected error: %v", err)
	}
	want := "Logged in as  on account  (role )."
	if got != want {
		t.Fatalf("loginConfirmation() = %q, want %q", got, want)
	}
}
