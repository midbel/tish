package tish

import (
	"strings"
	"testing"
)

func TestSimple_Expand(t *testing.T) {
	env := EmptyEnv()
	env.Define("FOO", "foo")
	env.Define("BAR", "bar")
	data := []struct {
		Input string
		Want  []string
	}{
		{
			Input: "echo foo bar",
			Want:  []string{"echo", "foo", "bar"},
		},
		{
			Input: "echo $FOO $BAR",
			Want:  []string{"echo", "foo", "bar"},
		},
		{
			Input: "echo ${#FOO}",
			Want:  []string{"echo", "3"},
		},
	}
	for _, d := range data {
		p, _ := NewParser(strings.NewReader(d.Input))
		c, err := p.Parse()
		if err != nil {
			t.Errorf("%s: error while parsing input: %s", d.Input, err)
			continue
		}
		s, ok := c.(Simple)
		if !ok {
			t.Errorf("%s: expected Simple, got %T", d.Input, c)
			continue
		}
		words := s.Expand(env)
		if len(words) != len(d.Want) {
			t.Errorf("%s: number of words mismatched! want %s, got %s", d.Input, d.Want, words)
		}
		for i := range d.Want {
			if d.Want[i] != words[i] {
				t.Errorf("%s: word mismatched! want %s, got %s", d.Input, d.Want[i], words[i])
			}
		}
	}
}

type WordCase struct {
	Word
	Want string
}

func TestExpand(t *testing.T) {
	t.Run("length", testLength)
	t.Run("replace", testReplace)
	t.Run("trim", testTrim)
	t.Run("transform", testTransform)
	t.Run("slice", testSlice)
}

func testLength(t *testing.T) {
	data := []WordCase{
		{
			Word: Length{ident: makeIdent("VAR")},
			Want: "6",
		},
		{
			Word: Length{ident: makeIdent("TEST")},
			Want: "0",
		},
		{
			Word: Length{ident: makeIdent("EMPTY")},
			Want: "0",
		},
	}
	testWordCase(t, data)
}

func testTrim(t *testing.T) {
	data := []WordCase{
		{
			Word: Trim{
				ident: makeIdent("VAR"),
				str:   makeIdent("bar"),
				part:  makeType(TokTrimSuffix),
			},
			Want: "foo",
		},
		{
			Word: Trim{
				ident: makeIdent("VAR"),
				str:   makeIdent("test"),
				part:  makeType(TokTrimSuffix),
			},
			Want: "foobar",
		},
		{
			Word: Trim{
				ident: makeIdent("VAR"),
				str:   makeIdent("test"),
				part:  makeType(TokTrimSuffixLong),
			},
			Want: "foobar",
		},
		{
			Word: Trim{
				ident: makeIdent("VAR"),
				str:   makeIdent("foo"),
				part:  makeType(TokTrimPrefix),
			},
			Want: "bar",
		},
		{
			Word: Trim{
				ident: makeIdent("VAR"),
				str:   makeIdent("test"),
				part:  makeType(TokTrimPrefix),
			},
			Want: "foobar",
		},
		{
			Word: Trim{
				ident: makeIdent("VAR"),
				str:   makeIdent("test"),
				part:  makeType(TokTrimPrefixLong),
			},
			Want: "foobar",
		},
	}
	testWordCase(t, data)
}

func testTransform(t *testing.T) {
	data := []WordCase{
		{
			Word: Transform{ident: makeIdent("VAR"), op: makeType(TokLower)},
			Want: "foobar",
		},
		{
			Word: Transform{ident: makeIdent("VAR"), op: makeType(TokLowerAll)},
			Want: "foobar",
		},
		{
			Word: Transform{ident: makeIdent("VAR"), op: makeType(TokUpper)},
			Want: "Foobar",
		},
		{
			Word: Transform{ident: makeIdent("VAR"), op: makeType(TokUpperAll)},
			Want: "FOOBAR",
		},
		{
			Word: Transform{ident: makeIdent("VAR"), op: makeType(TokReverse)},
			Want: "Foobar",
		},
		{
			Word: Transform{ident: makeIdent("VAR"), op: makeType(TokReverseAll)},
			Want: "FOOBAR",
		},
	}
	testWordCase(t, data)
}

func testReplace(t *testing.T) {
	data := []WordCase{
		{
			Word: Replace{
				ident: makeIdent("VAR"),
				src:   makeIdent("o"),
				dst:   makeIdent("-"),
				op:    makeType(TokReplace),
			},
			Want: "f-obar",
		},
		{
			Word: Replace{
				ident: makeIdent("VAR"),
				src:   makeIdent("o"),
				dst:   makeIdent("-"),
				op:    makeType(TokReplaceAll),
			},
			Want: "f--bar",
		},
		{
			Word: Replace{
				ident: makeIdent("VAR"),
				src:   makeIdent("bar"),
				dst:   makeIdent("foo"),
				op:    makeType(TokReplaceSuffix),
			},
			Want: "foofoo",
		},
		{
			Word: Replace{
				ident: makeIdent("VAR"),
				src:   makeIdent("---"),
				dst:   makeIdent("foo"),
				op:    makeType(TokReplaceSuffix),
			},
			Want: "foobar",
		},
		{
			Word: Replace{
				ident: makeIdent("VAR"),
				src:   makeIdent("foo"),
				dst:   makeIdent("bar"),
				op:    makeType(TokReplacePrefix),
			},
			Want: "barbar",
		},
		{
			Word: Replace{
				ident: makeIdent("VAR"),
				src:   makeIdent("---"),
				dst:   makeIdent("bar"),
				op:    makeType(TokReplacePrefix),
			},
			Want: "foobar",
		},
	}
	testWordCase(t, data)
}

func testSlice(t *testing.T) {
	data := []WordCase{
		{
			Word: Slice{
				ident:  makeIdent("VAR"),
				offset: Token{Literal: "1", Type: TokNumber},
				length: Token{Literal: "3", Type: TokNumber},
			},
			Want: "oob",
		},
		{
			Word: Slice{
				ident:  makeIdent("VAR"),
				offset: Token{Literal: "0", Type: TokNumber},
				length: Token{Literal: "3", Type: TokNumber},
			},
			Want: "foo",
		},
		{
			Word: Slice{
				ident:  makeIdent("VAR"),
				offset: Token{Literal: "1", Type: TokNumber},
				length: Token{Literal: "0", Type: TokNumber},
			},
			Want: "oobar",
		},
	}
	testWordCase(t, data)
}

func testWordCase(t *testing.T, data []WordCase) {
	e := makeEnv()
	for _, d := range data {
		got := d.Word.Expand(e)
		if got != d.Want {
			t.Errorf("%s: length mismatched! want %s, got %s", d.Word, d.Want, got)
		}
	}
}

func makeIdent(str string) Token {
	return Token{
		Literal: str,
		Type:    TokVariable,
	}
}

func makeType(k Kind) Token {
	tok := makeIdent("")
	tok.Type = k
	return tok
}

func makeEnv() Environment {
	e := EmptyEnv()
	e.Define("VAR", "foobar")
	e.Define("EMPTY", "")
	return e
}
