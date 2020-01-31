package tish

import (
	"testing"
)

func TestWords(t *testing.T) {
	data := []struct {
		Input string
		Words []string
	}{
		{
			Input: `echo`,
			Words: []string{"echo"},
		},
		{
			Input: `echo foo bar`,
			Words: []string{"echo", "foo", "bar"},
		},
		{
			Input: `echo foo\ bar`,
			Words: []string{"echo", "foo bar"},
		},
		{
			Input: `echo fo\o`,
			Words: []string{"echo", "foo"},
		},
		{
			Input: `echo "foobar"`,
			Words: []string{"echo", "foobar"},
		},
		{
			Input: `echo 'foobar'`,
			Words: []string{"echo", "foobar"},
		},
		{
			Input: `echo 'PWD=$PWD'`,
			Words: []string{"echo", "PWD=$PWD"},
		},
		{
			Input: `echo "foo bar"`,
			Words: []string{"echo", "foo bar"},
		},
		{
			Input: `echo "foo bar" "foo\" bar"`,
			Words: []string{"echo", "foo bar", "foo\" bar"},
		},
		{
			Input: `echo prefix" between "suffix`,
			Words: []string{"echo", "prefix between suffix"},
		},
		{
			Input: `echo prefix' between 'suffix`,
			Words: []string{"echo", "prefix between suffix"},
		},
	}
	for i, d := range data {
		s := NewScanner(d.Input)
		for tok, j := s.Scan(), 0; tok.Type != EOS; tok = s.Scan() {
			if j > len(d.Words) {
				t.Errorf("%d) too many tokens generated! want %d, got %d", i+1, len(d.Words), j)
				break
			}
			if tok.Type != Word {
				t.Errorf("%d) unexpected token type", i+1)
				break
			}
			if tok.Literal != d.Words[j] {
				t.Errorf("%d) unexpected token! want %q, got %q", i+1, d.Words[j], tok.Literal)
				break
			}
			j++
		}
	}
}
