package tish

import (
	"testing"
)

func TestWordExpand(t *testing.T) {
	p := NewEnvironment()
	p.Set("HOME", []string{"/home/midbel"})
	p.Set("SHELL", []string{"/bin/shell"})

	e := NewEnclosedEnvironment(p)
	e.Set("FOO", []string{"foo"})
	e.Set("BAR", []string{"bar"})

	data := []struct {
		Literal string
		Values  []string
	}{
		{Literal: "FOO", Values: []string{"foo"}},
		{Literal: "BAR", Values: []string{"bar"}},
		{Literal: "SHELL", Values: []string{"/bin/shell"}},
	}

	for _, d := range data {
		v := Variable(d.Literal)
		vs, err := v.Expand(e)
		if err != nil {
			t.Errorf("%s: unexpected error when expanding variable: %s", v, err)
			continue
		}
		if len(vs) != len(d.Values) {
			t.Errorf("%s: number of values mismatched: want %q, got %q", v, d.Values, vs)
			continue
		}
		for i := 0; i < len(vs); i++ {
			if vs[i] != d.Values[i] {
				t.Errorf("%s: mismatch value! want %s, got %s", v, d.Values[i], vs[i])
				break
			}
		}
		e.Del(d.Literal)
		if _, err := e.Get(d.Literal); err == nil {
			t.Errorf("%s: deleted variable has been resolved", v)
		}
	}
}
