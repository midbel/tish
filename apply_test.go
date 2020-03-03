package tish

import (
	"testing"
)

type ApplyCase struct {
	Apply
	Ident string
	Want  string
}

func TestApply(t *testing.T) {
	t.Run("identity", testApplyIdentity)
	t.Run("length", testApplyLength)
	t.Run("substring", testApplySubstring)
	t.Run("trim", testApplyTrim)
	t.Run("getorset", testApplyGetOrSet)
	t.Run("case", testApplyCase)
	t.Run("replace", testApplyReplace)
}

func testApplyReplace(t *testing.T) {
	var (
		ident = "STR"
		str   = "FOOBAR"
		env   = NewEnvironment()
	)
	env.Define(ident, []string{str})

	data := []ApplyCase{
		{
			Ident: ident,
			Want:  "F--BAR",
			Apply: Replace(Literal("OO"), Literal("--")),
		},
		{
			Ident: ident,
			Want:  "F--BAR",
			Apply: ReplaceAll(Literal("O"), Literal("-")),
		},
		{
			Ident: ident,
			Want:  str,
			Apply: Replace(Literal("foobar"), Literal("")),
		},
		{
			Ident: ident,
			Want:  str,
			Apply: ReplaceAll(Literal("foobar"), Literal("")),
		},
		{
			Ident: ident,
			Want:  str,
			Apply: ReplacePrefix(Literal("BAR"), Literal("FOO")),
		},
		{
			Ident: ident,
			Want:  "---BAR",
			Apply: ReplacePrefix(Literal("FOO"), Literal("---")),
		},
		{
			Ident: ident,
			Want:  "FOO---",
			Apply: ReplaceSuffix(Literal("BAR"), Literal("---")),
		},
	}

	runApplyTest(t, env, data...)
}

func testApplyCase(t *testing.T) {
	var (
		upper = "UPPER"
		lower = "LOWER"
		env   = NewEnvironment()
	)
	env.Define(upper, []string{"FOO BAR"})
	env.Define(lower, []string{"foo bar"})

	data := []ApplyCase{
		{Ident: lower, Want: "FOO BAR", Apply: Upper(true)},
		{Ident: lower, Want: "Foo bar", Apply: Upper(false)},
		{Ident: upper, Want: "foo bar", Apply: Lower(true)},
		{Ident: upper, Want: "fOO BAR", Apply: Lower(false)},
	}

	runApplyTest(t, env, data...)
}

func testApplyGetOrSet(t *testing.T) {
	var (
		ident = "STR"
		str   = "FOOBAR"
		env   = NewEnvironment()
	)

	env.Define(ident, []string{str})

	data := []ApplyCase{
		{
			Ident: "FOO",
			Want:  "FOO",
			Apply: SetIfUndef(Literal("FOO")),
		},
		{
			Ident: "BAR",
			Want:  "BAR",
			Apply: GetIfUndef(Literal("BAR")),
		},
		{
			Ident: ident,
			Want:  "BAR",
			Apply: GetIfDef(Literal("BAR")),
		},
	}

	runApplyTest(t, env, data...)

	if _, err := env.Resolve("FOO"); err != nil {
		t.Errorf("var 'FOO' should be set!")
	}
	if _, err := env.Resolve("BAR"); err == nil {
		t.Errorf("var 'BAR' should not be set!")
	}
}

func testApplyTrim(t *testing.T) {
	var (
		str    = "FOOBAR"
		prefix = "FOO"
		suffix = "BAR"
		ident  = "STR"
		env    = NewEnvironment()
	)
	env.Define(ident, []string{str})

	data := []ApplyCase{
		{
			Ident: ident,
			Want:  prefix,
			Apply: TrimSuffix(Literal(suffix), false),
		},
		{
			Ident: ident,
			Want:  suffix,
			Apply: TrimPrefix(Literal(prefix), false),
		},
		{
			Ident: ident,
			Want:  str,
			Apply: TrimSuffix(Literal(prefix), false),
		},
		{
			Ident: ident,
			Want:  str,
			Apply: TrimPrefix(Literal(suffix), false),
		},
		{
			Ident: "FOO",
			Want:  "FOOBAR",
			Apply: TrimPrefix(Literal("FOO"), false),
		},
		{
			Ident: "FOO",
			Want:  "BAR",
			Apply: TrimPrefix(Literal("FOO"), true),
		},
		{
			Ident: "BAR",
			Want:  "FOOBAR",
			Apply: TrimSuffix(Literal("BAR"), false),
		},
		{
			Ident: "BAR",
			Want:  "FOO",
			Apply: TrimSuffix(Literal("BAR"), true),
		},
	}

	env.Define("FOO", []string{"FOOFOOBAR"})
	env.Define("BAR", []string{"FOOBARBAR"})
	runApplyTest(t, env, data...)
}

func testApplyLength(t *testing.T) {
	var (
		n   = Length()
		env = NewEnvironment()
	)
	env.Define("FOO", []string{"FOO"})
	env.Define("BAR", []string{"BAR"})

	data := []ApplyCase{
		{Ident: "FOO", Want: "3", Apply: n},
		{Ident: "BAR", Want: "3", Apply: n},
	}

	runApplyTest(t, env, data...)
}

func testApplyIdentity(t *testing.T) {
	a := ApplyCase{
		Ident: "STR",
		Want:  "FOOBAR",
		Apply: Identity(),
	}

	env := NewEnvironment()
	env.Define(a.Ident, []string{a.Want})

	runApplyTest(t, env, a)
}

func testApplySubstring(t *testing.T) {
	var (
		ident = "STR"
		str   = "0123456789ABCDEF"
		env   = NewEnvironment()
	)
	env.Define(ident, []string{str})

	data := []ApplyCase{
		{
			Ident: ident,
			Want:  "",
			Apply: Substring(Number(0), Number(0)),
		},
		{
			Ident: ident,
			Want:  "0123",
			Apply: Substring(Number(0), Number(4)),
		},
		{
			Ident: ident,
			Want:  "4567",
			Apply: Substring(Number(4), Number(4)),
		},
		{
			Ident: ident,
			Want:  "456789",
			Apply: Substring(Number(4), Number(-6)),
		},
		{
			Ident: ident,
			Want:  "ABCDEF",
			Apply: Substring(Number(-6), Number(0)),
		},
		{
			Ident: ident,
			Want:  "ABCD",
			Apply: Substring(Number(-6), Number(4)),
		},
		{
			Ident: ident, Want: "ABCD",
			Apply: Substring(Number(-6), Number(-2)),
		},
		{
			Ident: ident,
			Want:  str,
			Apply: Substring(Number(-20), Number(0)),
		},
		{
			Ident: ident,
			Want:  str,
			Apply: Substring(Number(20), Number(0)),
		},
	}

	runApplyTest(t, env, data...)
}

func runApplyTest(t *testing.T, env *Env, data ...ApplyCase) {
	t.Helper()

	for i, d := range data {
		v := Variable{
			ident:  d.Ident,
			apply:  d.Apply,
			quoted: true,
		}
		vs, err := v.Expand(env)
		if err != nil {
			t.Errorf("%d) apply failure: %s", i+1, err)
			continue
		}
		if len(vs) == 0 && d.Want != "" {
			t.Errorf("%d) no values returned", i+1)
			continue
		}
		if d.Want == "" {
			if len(vs) > 0 {
				t.Errorf("%d) no values expected but got %q", i+1, vs)
			}
			continue
		}
		if vs[0] != d.Want {
			t.Errorf("%d) values mismatch! want %s, got %s", i+1, d.Want, vs[0])
		}
	}
}
