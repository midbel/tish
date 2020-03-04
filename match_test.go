package tish

import (
	"testing"
)

func TestMatch(t *testing.T) {
	data := []struct {
		Pattern string
		Input   string
		Match   bool
	}{
		{Pattern: "*", Input: "foobar", Match: true},
		{Pattern: "foobar", Input: "foobar", Match: true},
		{Pattern: "f??bar", Input: "foobar", Match: true},
		{Pattern: "fOObar", Input: "foobar", Match: false},
		{Pattern: "foo*", Input: "foobar", Match: true},
		{Pattern: "*bar", Input: "foobar", Match: true},
		{Pattern: "f*r", Input: "foobar", Match: true},
    {Pattern: "f*bar", Input: "foostar", Match: false},
		{Pattern: "f[a-z][a-z]bar", Input: "foobar", Match: true},
		{Pattern: "f[!0-9][^0-9]bar", Input: "foobar", Match: true},
		{Pattern: "f[oO][oO]bar", Input: "foobar", Match: true},
		{Pattern: "f[oO][oO]bar", Input: "fOObar", Match: true},
		{Pattern: "f[oO][oO]b[A-Z]r", Input: "foobar", Match: false},
	}
	for _, d := range data {
		got := Match(d.Input, d.Pattern)
		if got != d.Match {
			t.Errorf("%s(%s): invalid match! want %t, got %t", d.Input, d.Pattern, d.Match, got)
		}
	}
}
