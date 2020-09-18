package tish

import (
	"testing"
)

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

func testReplace(t *testing.T) {
	t.SkipNow()
}

func testTrim(t *testing.T) {
	t.SkipNow()
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

func testSlice(t *testing.T) {
	t.SkipNow()
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

func makeEnv() *Env {
	e := EmptyEnv()
	w := Literal{
		tokens: []Token{{Literal: "foobar", Type: TokLiteral}},
	}
	e.Define("VAR", w)
	e.Define("EMPTY", nil)
	return e
}
