package tish

import (
	"testing"
)

type StringListCase struct {
	Input  StringList
	Want   []string
	Cutset string
}

func TestStringList(t *testing.T) {
	t.Run("split", testStringListSplit)
	t.Run("fields", testStringListFields)
}

func testStringListSplit(t *testing.T) {
	data := []StringListCase{
		{
			Input: Quoted("foo bar"),
			Want:  []string{"foo bar"},
		},
		{
			Input:  Quoted("foo bar"),
			Want:   []string{"foo", "bar"},
			Cutset: " \t\n",
		},
		{
			Input:  Quoted("foo\nbar"),
			Want:   []string{"foo", "bar"},
			Cutset: " \t\n",
		},
		{
			Input:  Quoted("foo_<FOO>-<BAR>_bar"),
			Want:   []string{"foo", "<FOO>", "<BAR>", "bar"},
			Cutset: "_-",
		},
		{
			Input:  Unquoted("foo bar"),
			Want:   []string{"foo bar"},
			Cutset: "",
		},
		{
			Input:  Unquoted("foo.<FOO>\n<BAR>-bar"),
			Want:   []string{"foo", "<FOO>", "<BAR>", "bar"},
			Cutset: ".-\n",
		},
	}
	for j, d := range data {
		got := d.Input.Split(d.Cutset)
		if len(got) != len(d.Want) {
			t.Errorf("%d) %s: strings mismatched! want %q, got %q", j+1, d.Input, d.Want, got)
			continue
		}
		for i := range d.Want {
			if d.Want[i] != got[i] {
				t.Errorf("%d) %s: string mismatched! want %q, got %q", j+1, d.Input, d.Want[i], got[i])
			}
		}
	}
}

func testStringListFields(t *testing.T) {
	data := []StringListCase{
		{
			Input: Quoted("foo"),
			Want:  []string{"foo"},
		},
		{
			Input: Quoted("foo bar"),
			Want:  []string{"foo bar"},
		},
		{
			Input: Quoted("foo <FOO> <BAR> bar"),
			Want:  []string{"foo <FOO> <BAR> bar"},
		},
		{
			Input: Unquoted("foo"),
			Want:  []string{"foo"},
		},
		{
			Input: Unquoted("foo bar"),
			Want:  []string{"foo", "bar"},
		},
		{
			Input: Unquoted("foo <FOO> <BAR> bar"),
			Want:  []string{"foo", "<FOO>", "<BAR>", "bar"},
		},
	}
	for j, d := range data {
		got := d.Input.Fields()
		if len(got) != len(d.Want) {
			t.Errorf("%d) %s: strings mismatched! want %q, got %q", j+1, d.Input, d.Want, got)
			continue
		}
		for i := range d.Want {
			if d.Want[i] != got[i] {
				t.Errorf("%d) %s: string mismatched! want %q, got %q", j+1, d.Input, d.Want[i], got[i])
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
