package tish

import (
	"sort"
	"testing"
)

func TestWordExpand(t *testing.T) {
	t.Run("variables", testExpandVariables)
	t.Run("lists", testExpandLists)
	t.Run("expr", testExpandExpr)
	t.Run("braces", testExpandBraces)
	t.Run("assignments", testExpandAssignments)
	t.Run("words", testExpandWords)
}

func testExpandAssignments(t *testing.T) {
	data := []struct {
		ident  string
		values []string
	}{
		{ident: "FOO", values: []string{"FOOBAR"}},
	}
	for _, d := range data {
		env := NewEnvironment()
		env.Define(d.ident, d.values)

		vs, err := env.Resolve(d.ident)
		if err != nil {
			t.Errorf("%s: %s", d.ident, err)
			continue
		}
		if len(vs) != len(d.values) {
			t.Errorf("%s: mismatched values! want %q, got %q", d.ident, d.values, vs)
		}
		sort.Strings(vs)
		sort.Strings(d.values)
		for i := 0; i < len(vs); i++ {
			if vs[i] != d.values[i] {
				t.Errorf("%s: mismatched values! want %s, got %s", d.ident, d.values[i], vs[i])
			}
		}
	}
}

func testExpandBraces(t *testing.T) {
	env := buildEnv()
	data := []struct {
		Word
		Values []string
	}{
		{
			Word: Brace{
				word: makeList(kindSimple, Literal("foo"), Literal("bar")),
			},
			Values: []string{"foo", "bar"},
		},
		{
			Word: Brace{
				prolog: Literal("before-"),
				epilog: Literal("-after"),
				word:   makeList(kindSimple, Literal("foo"), Literal("bar")),
			},
			Values: []string{"before-foo-after", "before-bar-after"},
		},
		{
			Word: Brace{
				prolog: Literal("before-"),
				word:   makeList(kindSimple, Literal("foo"), Literal("bar")),
			},
			Values: []string{"before-foo", "before-bar"},
		},
		{
			Word: Brace{
				epilog: Literal("-after"),
				word:   makeList(kindSimple, Literal("foo"), Literal("bar")),
			},
			Values: []string{"foo-after", "bar-after"},
		},
		{
			Word: Brace{
				prolog: Brace{
					prolog: Literal("foo-"),
					word:   makeList(kindSimple, Literal("1"), Literal("2")),
				},
				word: Brace{
					prolog: Literal("-bar-"),
					word:   makeList(kindSimple, Literal("3"), Literal("4")),
				},
			},
			Values: []string{"foo-1-bar-3", "foo-1-bar-4", "foo-2-bar-3", "foo-2-bar-4"},
		},
		{
			Word: Brace{
				word: makeList(kindBraces,
					Brace{prolog: Literal("foo-"), word: makeList(kindBraces, Literal("1"), Literal("2"))},
					Brace{prolog: Literal("bar-"), word: makeList(kindBraces, Literal("3"), Literal("4"))},
				),
			},
			Values: []string{"foo-1", "foo-2", "bar-3", "bar-4"},
		},
	}
	for _, d := range data {
		vs, err := d.Word.Expand(env)
		if err != nil {
			t.Errorf("%s: fail to expand: %s", d.Word, err)
			continue
		}
		if len(vs) != len(d.Values) {
			t.Errorf("%s: mismatched values! want %q, got %q", d.Word, d.Values, vs)
		}
		sort.Strings(vs)
		sort.Strings(d.Values)
		for i := 0; i < len(vs); i++ {
			if vs[i] != d.Values[i] {
				t.Errorf("%s: mismatched value! want %s, got %s", d.Word, d.Values[i], vs[i])
			}
		}
	}
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
			Expr: Variable{ident: "NINE"},
			Want: 9,
		},
		{
			Expr: infix{
				left:  Number(1),
				right: Variable{ident: "NINE"},
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
		}
	}

	expr := infix{
		left:  Number(1),
		right: Variable{ident: "THREE"},
		op:    tokLeftShift,
	}
	e := Expr{expr: expr}
	got, err := e.Expand(env)
	if err != nil {
		t.Fatalf("%s: unexpected error when expanding: %s", e, err)
	}
	if len(got) != 1 {
		t.Fatalf("%s: unexpected number of result: %q", e, got)
	}
	if got[0] != "8" {
		t.Fatalf("%s: want %d, got %s", e, 1<<3, got[0])
	}
}

func testExpandLists(t *testing.T) {
	var (
		env  = buildEnv()
		list = List{
			words: []Word{
				Literal("echo"),
				Literal("foobar"),
				Variable{ident: "FOO"},
				Variable{ident: "BAR"},
				List{
					words: []Word{
						Literal("PWD"),
						Variable{ident: "PWD"},
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
		v := Variable{ident: d.Literal}
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
		if _, err := env.Resolve(d.Literal); err == nil {
			t.Errorf("%s: deleted variable has been resolved", v)
		}
	}
}

func testExpandWords(t *testing.T) {
	env := buildEnv()
	i := List{
		kind: kindWord,
		words: []Word{
			Literal("pre-"),
			Literal(" <"),
			Literal(" middle "),
			Literal("> "),
			Literal("-post"),
		},
	}
	want := "pre- < middle > -post"

	vs, err := i.Expand(env)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(vs) != 1 {
		t.Fatalf("values mismatched: %s", vs)
	}
	if vs[0] != want {
		t.Fatalf("values mismatched! want %s, got %s", want, vs[0])
	}
}

func buildEnv() *Env {
	p := NewEnvironment()
	p.Define("HOME", []string{"/home/midbel"})
	p.Define("SHELL", []string{"/bin/shell"})
	p.Define("PWD", []string{"github.com/midbel/tish"})
	p.Define("THREE", []string{"3"})

	e := NewEnclosedEnvironment(p)
	e.Define("FOO", []string{"foo"})
	e.Define("BAR", []string{"bar"})
	e.Define("NINE", []string{"9"})

	return e
}
