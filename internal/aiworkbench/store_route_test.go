package aiworkbench

import (
	"testing"
	"time"
)

func TestParseStoreRouteRejectsModelControlledIdentityAndInvalidCombinations(t *testing.T) {
	cases := []struct {
		input string
		valid bool
	}{
		{`{"kind":"STORE_OVERVIEW","status":"ALL_NON_DELETED"}`, true},
		{`{"kind":"STORE_OVERVIEW","status":"ACTIVE"}`, true},
		{`{"kind":"PRODUCT_TASK","status":""}`, true},
		{`{"kind":"CLARIFY","status":""}`, true},
		{`{"kind":"STORE_OVERVIEW","status":"ALL_NON_DELETED","organization":"other"}`, false},
		{`{"kind":"STORE_OVERVIEW","status":"DISABLED"}`, false},
		{`{"kind":"PRODUCT_TASK","status":"ACTIVE"}`, false},
		{`{"kind":"CLARIFY","status":null}`, false},
		{`{"kind":"CLARIFY"}`, false},
		{`{"kind":"STORE_OVERVIEW","kind":"CLARIFY","status":""}`, false},
		{`{"kind":"STORE_OVERVIEW","status":"ALL_NON_DELETED"} trailing`, false},
	}
	for _, tc := range cases {
		_, err := ParseStoreRoute([]byte(tc.input))
		if (err == nil) != tc.valid {
			t.Errorf("%s: %v", tc.input, err)
		}
	}
}

func TestRouteCommitAndLostFence(t *testing.T) {
	deadline := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name         string
		now          time.Time
		settled      bool
		commit, lost bool
	}{
		{"before cutoff", deadline.Add(-time.Nanosecond), true, true, false},
		{"at cutoff", deadline, true, true, false},
		{"after cutoff settled", deadline.Add(time.Nanosecond), true, false, true},
		{"after cutoff unknown", deadline.Add(time.Nanosecond), false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := RouteCommitEligible(tc.now, deadline); got != tc.commit {
				t.Fatalf("commit=%v", got)
			}
			if got := RouteLostEligible(tc.now, deadline, tc.settled); got != tc.lost {
				t.Fatalf("lost=%v", got)
			}
		})
	}
}
