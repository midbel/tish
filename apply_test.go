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
	env.Set(ident, []string{str})

	data := []ApplyCase{
		{Ident: ident, Want: "F--BAR", Apply: Replace("OO", "--")},
		{Ident: ident, Want: "F--BAR", Apply: ReplaceAll("O", "-")},
		{Ident: ident, Want: str, Apply: Replace("foobar", "")},
		{Ident: ident, Want: str, Apply: ReplaceAll("foobar", "")},
		{Ident: ident, Want: str, Apply: ReplacePrefix("BAR", "FOO")},
		{Ident: ident, Want: "---BAR", Apply: ReplacePrefix("FOO", "---")},
		{Ident: ident, Want: "FOO---", Apply: ReplaceSuffix("BAR", "---")},
	}

	runApplyTest(t, env, data...)
}

func testApplyCase(t *testing.T) {
	var (
		upper = "UPPER"
		lower = "LOWER"
		env   = NewEnvironment()
	)
	env.Set(upper, []string{"FOO BAR"})
	env.Set(lower, []string{"foo bar"})

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

	env.Set(ident, []string{str})

	data := []ApplyCase{
		{Ident: "FOO", Want: "FOO", Apply: SetIfUndef("FOO")},
		{Ident: "BAR", Want: "BAR", Apply: GetIfUndef("BAR")},
		{Ident: ident, Want: "BAR", Apply: GetIfDef("BAR")},
	}

	runApplyTest(t, env, data...)

	if _, err := env.Get("FOO"); err != nil {
		t.Errorf("var 'FOO' should be set!")
	}
	if _, err := env.Get("BAR"); err == nil {
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
	env.Set(ident, []string{str})

	data := []ApplyCase{
		{Ident: ident, Want: prefix, Apply: TrimSuffix(suffix, false)},
		{Ident: ident, Want: suffix, Apply: TrimPrefix(prefix, false)},
		{Ident: ident, Want: str, Apply: TrimSuffix(prefix, false)},
		{Ident: ident, Want: str, Apply: TrimPrefix(suffix, false)},
		{Ident: "FOO", Want: "FOOBAR", Apply: TrimPrefix("FOO", false)},
		{Ident: "FOO", Want: "BAR", Apply: TrimPrefix("FOO", true)},
		{Ident: "BAR", Want: "FOOBAR", Apply: TrimSuffix("BAR", false)},
		{Ident: "BAR", Want: "FOO", Apply: TrimSuffix("BAR", true)},
	}

	env.Set("FOO", []string{"FOOFOOBAR"})
	env.Set("BAR", []string{"FOOBARBAR"})
	runApplyTest(t, env, data...)
}

func testApplyLength(t *testing.T) {
	var (
		n   = Length()
		env = NewEnvironment()
	)
	env.Set("FOO", []string{"FOO"})
	env.Set("BAR", []string{"BAR"})

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
	env.Set(a.Ident, []string{a.Want})

	runApplyTest(t, env, a)
}

func testApplySubstring(t *testing.T) {
	var (
		ident = "STR"
		str   = "0123456789ABCDEF"
		env   = NewEnvironment()
	)
	env.Set(ident, []string{str})

	data := []ApplyCase{
		{Ident: ident, Want: str, Apply: Substring(0, 0)},
		{Ident: ident, Want: "0123", Apply: Substring(0, 4)},
		{Ident: ident, Want: "4567", Apply: Substring(4, 4)},
		{Ident: ident, Want: "456789", Apply: Substring(4, -6)},
		{Ident: ident, Want: "ABCDEF", Apply: Substring(-6, 0)},
		{Ident: ident, Want: "ABCD", Apply: Substring(-6, 4)},
		{Ident: ident, Want: "ABCD", Apply: Substring(-6, -2)},
		{Ident: ident, Want: str, Apply: Substring(-20, 0)},
		{Ident: ident, Want: str, Apply: Substring(20, 0)},
	}

	runApplyTest(t, env, data...)
}

func runApplyTest(t *testing.T, env *Env, data ...ApplyCase) {
	t.Helper()

	for i, d := range data {
		v := Variable{
			ident: d.Ident,
			apply: d.Apply,
		}
		vs, err := v.Expand(env)
		if err != nil {
			t.Errorf("%d) apply failure: %s", i+1, err)
			continue
		}
		if len(vs) == 0 {
			t.Errorf("%d) no values returned", i+1)
			continue
		}
		if vs[0] != d.Want {
			t.Errorf("%d) values mismatch! want %s, got %s", i+1, d.Want, vs[0])
		}
	}
}
