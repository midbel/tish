package tish

import (
	"reflect"
	"testing"
)

type WordCase struct {
	Word
	Want []string
}

func TestWord(t *testing.T) {
	data := []WordCase{
		{
			Word: createLiteral(createQuotedToken("<foo bar>", false, TokLiteral)),
			Want: []string{"<foo", "bar>"},
		},
		{
			Word: createLiteral(createQuotedToken("-foo bar-", true, TokLiteral)),
			Want: []string{"-foo bar-"},
		},
		{
			Word: createLiteral(createQuotedToken("SPACE", false, TokVariable)),
			Want: []string{"foo", "bar"},
		},
		{
			Word: createLiteral(createQuotedToken("SPACE", true, TokVariable)),
			Want: []string{"foo bar"},
		},
		{
			Word: WordList{
				words: []Word{
					createLiteral(createQuotedToken("SPACE", false, TokVariable)),
					createLiteral(createQuotedToken("-test-", true, TokLiteral)),
					createLiteral(createQuotedToken("SPACE", false, TokVariable)),
				},
			},
			Want: []string{"foo", "bar-test-foo", "bar"},
		},
		{
			Word: WordList{
				words: []Word{
					createLiteral(createQuotedToken("SPACE", true, TokVariable)),
					createLiteral(createQuotedToken("-test-", true, TokLiteral)),
					createLiteral(createQuotedToken("SPACE", true, TokVariable)),
				},
			},
			Want: []string{"foo bar-test-foo bar"},
		},
		{
			Word: WordList{
				words: []Word{
					createLiteral(createQuotedToken("<begin> ", true, TokLiteral)),
					createLiteral(createQuotedToken("SPACE", true, TokVariable)),
					createLiteral(createQuotedToken("-test-", true, TokLiteral)),
					createLiteral(createQuotedToken("SPACE", true, TokVariable)),
					createLiteral(createQuotedToken(" <end>", true, TokLiteral)),
				},
			},
			Want: []string{"<begin> foo bar-test-foo bar <end>"},
		},
		{
			Word: WordList{
				words: []Word{
					createLiteral(createQuotedToken("SPACE", false, TokVariable)),
					createLiteral(createQuotedToken("SPACE", false, TokVariable)),
				},
			},
			Want: []string{"foo", "barfoo", "bar"},
		},
		{
			Word: WordList{
				words: []Word{
					Serie{
						words: []Word{
							createLiteral(createQuotedToken("A", false, TokLiteral)),
							createLiteral(createQuotedToken("B", false, TokLiteral)),
							createLiteral(createQuotedToken("C", false, TokLiteral)),
						},
					},
					Serie{
						words: []Word{
							createLiteral(createQuotedToken("1", false, TokLiteral)),
							createLiteral(createQuotedToken("2", false, TokLiteral)),
							createLiteral(createQuotedToken("3", false, TokLiteral)),
						},
					},
				},
			},
			Want: []string{"A1", "A2", "A3", "B1", "B2", "B3", "C1", "C2", "C3"},
		},
		{
			Word: WordList{
				words: []Word{
					Serie{
						words: []Word{
							createLiteral(createQuotedToken("A", false, TokLiteral)),
							createLiteral(createQuotedToken("B", false, TokLiteral)),
							createLiteral(createQuotedToken("C", false, TokLiteral)),
						},
					},
					Range{
						first: createLiteral(createQuotedToken("1", false, TokNumber)),
						last:  createLiteral(createQuotedToken("3", false, TokNumber)),
						incr:  createLiteral(createQuotedToken("1", false, TokNumber)),
					},
				},
			},
			Want: []string{"A1", "A2", "A3", "B1", "B2", "B3", "C1", "C2", "C3"},
		},
		{
			Word: WordList{
				words: []Word{
					createLiteral(createQuotedToken("SPACE", false, TokVariable)),
					createLiteral(createQuotedToken("-", false, TokLiteral)),
					Serie{
						words: []Word{
							createLiteral(createQuotedToken("A", false, TokLiteral)),
							createLiteral(createQuotedToken("B", false, TokLiteral)),
							createLiteral(createQuotedToken("C", false, TokLiteral)),
						},
					},
					Serie{
						words: []Word{
							createLiteral(createQuotedToken("1", false, TokLiteral)),
							createLiteral(createQuotedToken("2", false, TokLiteral)),
							createLiteral(createQuotedToken("3", false, TokLiteral)),
						},
					},
					createLiteral(createQuotedToken("-", false, TokLiteral)),
					createLiteral(createQuotedToken("SPACE", false, TokVariable)),
				},
			},
			Want: []string{
				"foo", "bar-A1-foo", "bar",
				"foo", "bar-A2-foo", "bar",
				"foo", "bar-A3-foo", "bar",
				"foo", "bar-B1-foo", "bar",
				"foo", "bar-B2-foo", "bar",
				"foo", "bar-B3-foo", "bar",
				"foo", "bar-C1-foo", "bar",
				"foo", "bar-C2-foo", "bar",
				"foo", "bar-C3-foo", "bar",
			},
		},
	}
	testWordCase(t, data)
}

func TestExpand(t *testing.T) {
	t.Run("length", testLength)
	t.Run("replace", testReplace)
	t.Run("trim", testTrim)
	t.Run("transform", testTransform)
	t.Run("slice", testSlice)
	t.Run("expr", testExpr)
}

func testLength(t *testing.T) {
	data := []WordCase{
		{
			Word: Length{ident: createToken("VAR", TokVariable)},
			Want: []string{"6"},
		},
		{
			Word: Length{ident: createToken("TEST", TokVariable)},
			Want: []string{"0"},
		},
		{
			Word: Length{ident: createToken("EMPTY", TokVariable)},
			Want: []string{"0"},
		},
	}
	testWordCase(t, data)
}

func testTrim(t *testing.T) {
	data := []WordCase{
		{
			Word: Trim{
				ident: createToken("VAR", TokVariable),
				str:   createToken("bar", TokLiteral),
				part:  createType(TokTrimSuffix),
			},
			Want: []string{"foo"},
		},
		{
			Word: Trim{
				ident: createToken("VAR", TokVariable),
				str:   createToken("test", TokLiteral),
				part:  createType(TokTrimSuffix),
			},
			Want: []string{"foobar"},
		},
		{
			Word: Trim{
				ident: createToken("VAR", TokVariable),
				str:   createToken("test", TokLiteral),
				part:  createType(TokTrimSuffixLong),
			},
			Want: []string{"foobar"},
		},
		{
			Word: Trim{
				ident: createToken("VAR", TokVariable),
				str:   createToken("foo", TokLiteral),
				part:  createType(TokTrimPrefix),
			},
			Want: []string{"bar"},
		},
		{
			Word: Trim{
				ident: createToken("VAR", TokVariable),
				str:   createToken("test", TokLiteral),
				part:  createType(TokTrimPrefix),
			},
			Want: []string{"foobar"},
		},
		{
			Word: Trim{
				ident: createToken("VAR", TokVariable),
				str:   createToken("test", TokLiteral),
				part:  createType(TokTrimPrefixLong),
			},
			Want: []string{"foobar"},
		},
	}
	testWordCase(t, data)
}

func testTransform(t *testing.T) {
	data := []WordCase{
		{
			Word: Transform{
				ident: createToken("VAR", TokVariable),
				op:    createType(TokLower),
			},
			Want: []string{"foobar"},
		},
		{
			Word: Transform{
				ident: createToken("VAR", TokVariable),
				op:    createType(TokLowerAll),
			},
			Want: []string{"foobar"},
		},
		{
			Word: Transform{
				ident: createToken("VAR", TokVariable),
				op:    createType(TokUpper),
			},
			Want: []string{"Foobar"},
		},
		{
			Word: Transform{
				ident: createToken("VAR", TokVariable),
				op:    createType(TokUpperAll),
			},
			Want: []string{"FOOBAR"},
		},
		{
			Word: Transform{
				ident: createToken("VAR", TokVariable),
				op:    createType(TokReverse),
			},
			Want: []string{"Foobar"},
		},
		{
			Word: Transform{
				ident: createToken("VAR", TokVariable),
				op:    createType(TokReverseAll),
			},
			Want: []string{"FOOBAR"},
		},
	}
	testWordCase(t, data)
}

func testReplace(t *testing.T) {
	data := []WordCase{
		{
			Word: Replace{
				ident: createToken("VAR", TokVariable),
				src:   createToken("o", TokLiteral),
				dst:   createToken("-", TokLiteral),
				op:    createType(TokReplace),
			},
			Want: []string{"f-obar"},
		},
		{
			Word: Replace{
				ident: createToken("VAR", TokVariable),
				src:   createToken("o", TokLiteral),
				dst:   createToken("-", TokLiteral),
				op:    createType(TokReplaceAll),
			},
			Want: []string{"f--bar"},
		},
		{
			Word: Replace{
				ident: createToken("VAR", TokVariable),
				src:   createToken("bar", TokLiteral),
				dst:   createToken("foo", TokLiteral),
				op:    createType(TokReplaceSuffix),
			},
			Want: []string{"foofoo"},
		},
		{
			Word: Replace{
				ident: createToken("VAR", TokVariable),
				src:   createToken("---", TokLiteral),
				dst:   createToken("foo", TokLiteral),
				op:    createType(TokReplaceSuffix),
			},
			Want: []string{"foobar"},
		},
		{
			Word: Replace{
				ident: createToken("VAR", TokVariable),
				src:   createToken("foo", TokLiteral),
				dst:   createToken("bar", TokLiteral),
				op:    createType(TokReplacePrefix),
			},
			Want: []string{"barbar"},
		},
		{
			Word: Replace{
				ident: createToken("VAR", TokVariable),
				src:   createToken("---", TokLiteral),
				dst:   createToken("bar", TokLiteral),
				op:    createType(TokReplacePrefix),
			},
			Want: []string{"foobar"},
		},
	}
	testWordCase(t, data)
}

func testSlice(t *testing.T) {
	data := []WordCase{
		{
			Word: Slice{
				ident:  createToken("VAR", TokVariable),
				offset: createToken("1", TokNumber),
				length: createToken("3", TokNumber),
			},
			Want: []string{"oob"},
		},
		{
			Word: Slice{
				ident:  createToken("VAR", TokVariable),
				offset: createToken("0", TokNumber),
				length: createToken("3", TokNumber),
			},
			Want: []string{"foo"},
		},
		{
			Word: Slice{
				ident:  createToken("VAR", TokVariable),
				offset: createToken("1", TokNumber),
				length: createToken("0", TokNumber),
			},
			Want: []string{"oobar"},
		},
	}
	testWordCase(t, data)
}

func testWordCase(t *testing.T, data []WordCase) {
	env := makeEnv()
	for _, d := range data {
		ex := createExpander(d.Word)
		got := ex.Expand(env)
		if !reflect.DeepEqual(d.Want, got) {
			t.Errorf("%s: words mismatched! want %q, got %q", d.Word, d.Want, got)
		}
	}
}

func testExpr(t *testing.T) {
	data := []struct {
		Eval Evaluator
		Want int
	}{
		{
			Eval: Prefix{
				op:    TokSub,
				right: createNumber(createToken("1", TokNumber)),
			},
			Want: -1,
		},
		{
			Eval: Prefix{
				op:    TokBinNot,
				right: createNumber(createToken("2", TokNumber)),
			},
			Want: -3,
		},
		{
			Eval: Infix{
				op:   TokAdd,
				left: createNumber(createToken("1", TokNumber)),
				right: Prefix{
					op:    TokSub,
					right: createNumber(createToken("1", TokNumber)),
				},
			},
			Want: 0,
		},
		{
			Eval: Infix{
				op:    TokSub,
				left:  createIdentifier(createToken("VAR", TokVariable)),
				right: createNumber(createToken("1", TokNumber)),
			},
			Want: 0,
		},
	}
	env := EmptyEnv()
	env.Define("VAR", "1")
	for _, d := range data {
		got, err := d.Eval.Eval(env)
		if err != nil {
			t.Errorf("%s: unexpected error: %s", d.Eval, err)
			continue
		}
		if got != d.Want {
			t.Errorf("%s: result mismatched! want %d, got %d", d.Eval, d.Want, got)
		}
	}
}

func makeEnv() Environment {
	e := EmptyEnv()
	e.Define(IFS, " \t")
	e.Define("VAR", "foobar")
	e.Define("SPACE", "foo bar")
	e.Define("EMPTY", "")
	return e
}
