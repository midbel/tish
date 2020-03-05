package tish

import (
	"testing"
)

func TestSplit(t *testing.T) {
	data := []struct {
		Input string
		Want  []string
	}{
		{
			Input: `foobar`,
			Want:  []string{"foobar"},
		},
		{
			Input: `foo      bar`,
			Want:  []string{"foo", "bar"},
		},
		{
			Input: "foo\n\nbar",
			Want:  []string{"foo", "bar"},
		},
		{
			Input: `'foo' "bar"`,
			Want:  []string{"'foo'", "\"bar\""},
		},
		{
			Input: ``,
			Want:  []string{""},
		},
	}
	for _, d := range data {
		got := Split(d.Input)
		if len(got) != len(d.Want) {
			t.Errorf("%s: length mismatched! want %d, got %d (%q)", d.Input, len(d.Want), len(got), got)
			continue
		}
		for i := 0; i < len(got); i++ {
			if got[i] != d.Want[i] {
				t.Errorf("%s: values mismatched! want %s, got %s", d.Input, d.Want[i], got[i])
			}
		}
	}
}
