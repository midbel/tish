package tish

import (
	"strings"
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
	t.Run("expr", testExpr)
	t.Run("series", testSeries)
	t.Run("ranges", testRanges)
}

func testSeries(t *testing.T) {
	data := []WordCase{
		{
			Word: Serie{
				words: []Word{
					createLiteral(createToken("foo", TokLiteral)),
					createLiteral(createToken("bar", TokLiteral)),
				},
			},
			Want: "foo bar",
		},
		{
			Word: Serie{
				prefix: createLiteral(createToken("prefix-", TokLiteral)),
				suffix: createLiteral(createToken("-suffix", TokLiteral)),
				words: []Word{
					createLiteral(createToken("foo", TokLiteral)),
					createLiteral(createToken("bar", TokLiteral)),
				},
			},
			Want: "prefix-foo-suffix prefix-bar-suffix",
		},
		{
			Word: Serie{
				words: []Word{
					Serie{
						words: []Word{
							createLiteral(createToken("A", TokLiteral)),
							createLiteral(createToken("B", TokLiteral)),
							createLiteral(createToken("C", TokLiteral)),
						},
					},
					Serie{
						words: []Word{
							createLiteral(createToken("a", TokLiteral)),
							createLiteral(createToken("b", TokLiteral)),
							createLiteral(createToken("c", TokLiteral)),
						},
					},
				},
			},
			Want: "A B C a b c",
		},
		{
			Word: Serie{
				words: []Word{
					createLiteral(createToken("A", TokLiteral)),
					createLiteral(createToken("B", TokLiteral)),
					createLiteral(createToken("C", TokLiteral)),
				},
				suffix: Serie{
					words: []Word{
						createLiteral(createToken("1", TokLiteral)),
						createLiteral(createToken("2", TokLiteral)),
						createLiteral(createToken("3", TokLiteral)),
					},
				},
			},
			Want: "A1 A2 A3 B1 B2 B3 C1 C2 C3",
		},
	}
	testWordCase(t, data)
}

func testRanges(t *testing.T) {
	data := []WordCase{
		{
			Word: Range{
				first: createLiteral(createToken("0", TokNumber)),
				last:  createLiteral(createToken("5", TokNumber)),
				incr:  createLiteral(createToken("1", TokNumber)),
			},
			Want: "0 1 2 3 4 5",
		},
		{
			Word: Range{
				first: createLiteral(createToken("005", TokNumber)),
				last:  createLiteral(createToken("10", TokNumber)),
				incr:  createLiteral(createToken("1", TokNumber)),
			},
			Want: "005 006 007 008 009 010",
		},
		{
			Word: Range{
				first: createLiteral(createToken("5", TokNumber)),
				last:  createLiteral(createToken("0", TokNumber)),
				incr:  createLiteral(createToken("1", TokNumber)),
			},
			Want: "5 4 3 2 1 0",
		},
		{
			Word: Range{
				first: createLiteral(createToken("0", TokNumber)),
				last:  createLiteral(createToken("10", TokNumber)),
				incr:  createLiteral(createToken("2", TokNumber)),
			},
			Want: "0 2 4 6 8 10",
		},
		{
			Word: Range{
				first: createLiteral(createToken("5", TokNumber)),
				last:  createLiteral(createToken("5", TokNumber)),
				incr:  createLiteral(createToken("2", TokNumber)),
			},
			Want: "5",
		},
		{
			Word: Range{
				first: createLiteral(createToken("0", TokNumber)),
				last:  createLiteral(createToken("0", TokNumber)),
				incr:  createLiteral(createToken("2", TokNumber)),
			},
			Want: "0",
		},
		{
			Word: Range{
				prefix: createLiteral(createToken("foo-", TokLiteral)),
				suffix: createLiteral(createToken("-bar", TokLiteral)),
				first:  createLiteral(createToken("0", TokNumber)),
				last:   createLiteral(createToken("10", TokNumber)),
				incr:   createLiteral(createToken("2", TokNumber)),
			},
			Want: "foo-0-bar foo-2-bar foo-4-bar foo-6-bar foo-8-bar foo-10-bar",
		},
	}
	testWordCase(t, data)
}

func testLength(t *testing.T) {
	data := []WordCase{
		{
			Word: Length{ident: createToken("VAR", TokVariable)},
			Want: "6",
		},
		{
			Word: Length{ident: createToken("TEST", TokVariable)},
			Want: "0",
		},
		{
			Word: Length{ident: createToken("EMPTY", TokVariable)},
			Want: "0",
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
			Want: "foo",
		},
		{
			Word: Trim{
				ident: createToken("VAR", TokVariable),
				str:   createToken("test", TokLiteral),
				part:  createType(TokTrimSuffix),
			},
			Want: "foobar",
		},
		{
			Word: Trim{
				ident: createToken("VAR", TokVariable),
				str:   createToken("test", TokLiteral),
				part:  createType(TokTrimSuffixLong),
			},
			Want: "foobar",
		},
		{
			Word: Trim{
				ident: createToken("VAR", TokVariable),
				str:   createToken("foo", TokLiteral),
				part:  createType(TokTrimPrefix),
			},
			Want: "bar",
		},
		{
			Word: Trim{
				ident: createToken("VAR", TokVariable),
				str:   createToken("test", TokLiteral),
				part:  createType(TokTrimPrefix),
			},
			Want: "foobar",
		},
		{
			Word: Trim{
				ident: createToken("VAR", TokVariable),
				str:   createToken("test", TokLiteral),
				part:  createType(TokTrimPrefixLong),
			},
			Want: "foobar",
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
			Want: "foobar",
		},
		{
			Word: Transform{
				ident: createToken("VAR", TokVariable),
				op:    createType(TokLowerAll),
			},
			Want: "foobar",
		},
		{
			Word: Transform{
				ident: createToken("VAR", TokVariable),
				op:    createType(TokUpper),
			},
			Want: "Foobar",
		},
		{
			Word: Transform{
				ident: createToken("VAR", TokVariable),
				op:    createType(TokUpperAll),
			},
			Want: "FOOBAR",
		},
		{
			Word: Transform{
				ident: createToken("VAR", TokVariable),
				op:    createType(TokReverse),
			},
			Want: "Foobar",
		},
		{
			Word: Transform{
				ident: createToken("VAR", TokVariable),
				op:    createType(TokReverseAll),
			},
			Want: "FOOBAR",
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
			Want: "f-obar",
		},
		{
			Word: Replace{
				ident: createToken("VAR", TokVariable),
				src:   createToken("o", TokLiteral),
				dst:   createToken("-", TokLiteral),
				op:    createType(TokReplaceAll),
			},
			Want: "f--bar",
		},
		{
			Word: Replace{
				ident: createToken("VAR", TokVariable),
				src:   createToken("bar", TokLiteral),
				dst:   createToken("foo", TokLiteral),
				op:    createType(TokReplaceSuffix),
			},
			Want: "foofoo",
		},
		{
			Word: Replace{
				ident: createToken("VAR", TokVariable),
				src:   createToken("---", TokLiteral),
				dst:   createToken("foo", TokLiteral),
				op:    createType(TokReplaceSuffix),
			},
			Want: "foobar",
		},
		{
			Word: Replace{
				ident: createToken("VAR", TokVariable),
				src:   createToken("foo", TokLiteral),
				dst:   createToken("bar", TokLiteral),
				op:    createType(TokReplacePrefix),
			},
			Want: "barbar",
		},
		{
			Word: Replace{
				ident: createToken("VAR", TokVariable),
				src:   createToken("---", TokLiteral),
				dst:   createToken("bar", TokLiteral),
				op:    createType(TokReplacePrefix),
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
				ident:  createToken("VAR", TokVariable),
				offset: createToken("1", TokNumber),
				length: createToken("3", TokNumber),
			},
			Want: "oob",
		},
		{
			Word: Slice{
				ident:  createToken("VAR", TokVariable),
				offset: createToken("0", TokNumber),
				length: createToken("3", TokNumber),
			},
			Want: "foo",
		},
		{
			Word: Slice{
				ident:  createToken("VAR", TokVariable),
				offset: createToken("1", TokNumber),
				length: createToken("0", TokNumber),
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
		if strings.Join(got, " ") != d.Want {
			t.Errorf("%s: length mismatched! want %s, got %s", d.Word, d.Want, got)
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
	e.Define("VAR", "foobar")
	e.Define("EMPTY", "")
	return e
}
