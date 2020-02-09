package tish

import (
	"testing"
)

func TestWordExpand(t *testing.T) {
	t.Run("variables", testExpandVariables)
	t.Run("lists", testExpandLists)
	t.Run("expr", testExpandExpr)
}

func testExpandExpr(t *testing.T) {
	env := buildEnv()

	data := []struct {
		Expr Evaluator
		Want Number
	}{
		{
			Expr: Number(1),
			Want: 1,
		},
		{
			Expr: Variable("NINE"),
			Want: 9,
		},
		{
			Expr: infix{
				left:  Number(1),
				right: Variable("NINE"),
				op:    plus,
			},
			Want: 10,
		},
	}
	for _, d := range data {
		got, err := d.Expr.Eval(env)
		if err != nil {
			t.Errorf("%s: unexpected error when expanding expr: %s", d.Expr, err)
			continue
		}
		if got != d.Want {
			t.Errorf("%s: mismatch value! want %s, got %s", d.Expr, d.Want, got)
			break
		}
	}
}

func testExpandLists(t *testing.T) {
	var (
		env  = buildEnv()
		list = List{
			words: []Word{
				Literal("echo"),
				Literal("foobar"),
				Variable("FOO"),
				Variable("BAR"),
				List{
					words: []Word{
						Literal("PWD"),
						Variable("PWD"),
					},
				},
			},
		}
		values = []string{"echo", "foobar", "foo", "bar", "PWD", "github.com/midbel/tish"}
	)

	vs, err := list.Expand(env)
	if err != nil {
		t.Errorf("unexpeted error: %s", err)
		return
	}
	if len(vs) != len(values) {
		t.Errorf("unexpected number of values! want %q, got %q", values, vs)
		return
	}
	for i := 0; i < len(vs); i++ {
		if vs[i] != values[i] {
			t.Errorf("unexpected value! want %s, got %s", values[i], vs[i])
		}
	}
}

func testExpandVariables(t *testing.T) {
	env := buildEnv()

	data := []struct {
		Literal string
		Values  []string
		Defined bool
	}{
		{
			Literal: "FOO",
			Values:  []string{"foo"},
			Defined: true,
		},
		{
			Literal: "BAR",
			Values:  []string{"bar"},
			Defined: true,
		},
		{
			Literal: "SHELL",
			Values:  []string{"/bin/shell"},
			Defined: true,
		},
		{
			Literal: "MAIL",
			Values:  []string{},
			Defined: false,
		},
	}

	for _, d := range data {
		v := Variable(d.Literal)
		vs, err := v.Expand(env)
		if d.Defined {
			if err != nil {
				t.Errorf("%s: unexpected error when expanding variable: %s", v, err)
				continue
			}
		} else {
			if err == nil {
				t.Errorf("%s: variable not defined has been resolved", v)
			}
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
		env.Del(d.Literal)
		if _, err := env.Get(d.Literal); err == nil {
			t.Errorf("%s: deleted variable has been resolved", v)
		}
	}
}

func buildEnv() *Env {
	p := NewEnvironment()
	p.Set("HOME", []string{"/home/midbel"})
	p.Set("SHELL", []string{"/bin/shell"})
	p.Set("PWD", []string{"github.com/midbel/tish"})

	e := NewEnclosedEnvironment(p)
	e.Set("FOO", []string{"foo"})
	e.Set("BAR", []string{"bar"})
	e.Set("NINE", []string{"9"})

	return e
}
