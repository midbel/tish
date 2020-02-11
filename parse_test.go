package tish

import (
	"testing"
)

type ParseCase struct {
	Input string
	Word  Word
}

func TestParse(t *testing.T) {
	t.Run("simple", testParseSimple)
	t.Run("substitution", testParseSubstitution)
	t.Run("arithmetic", testParseArithmetic)
	t.Run("braces", testParseBraces)
}

func testParseBraces(t *testing.T) {
	data := []ParseCase{
		{
			Input: `echo {foo,bar}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Brace{
					word: makeList(kindBraces, Literal("foo"), Literal("bar")),
				},
			),
		},
		{
			Input: `echo {foobar}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Literal("{foobar}"),
			),
		},
		{
			Input: `echo {}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Literal("{}"),
			),
		},
		{
			Input: `echo {foo-{1,2}, bar-{3,4}}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Brace{
					word: makeList(kindBraces,
						Brace{
							prolog: Literal("foo-"),
							word:   makeList(kindBraces, Literal("1"), Literal("2")),
						},
						Brace{
							prolog: Literal("bar-"),
							word:   makeList(kindBraces, Literal("3"), Literal("4")),
						},
					),
				},
			),
		},
		{
			Input: `echo prolog-{foo,bar}-epilog`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Brace{
					prolog: Literal("prolog-"),
					epilog: Literal("-epilog"),
					word:   makeList(kindBraces, Literal("foo"), Literal("bar")),
				},
			),
		},
		{
			Input: `echo foo-{1,2}-bar-{3,4}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Brace{
					prolog: Brace{
						prolog: Literal("foo-"),
						word:   makeList(kindBraces, Literal("1"), Literal("2")),
					},
					word: Brace{
						prolog: Literal("-bar-"),
						word:   makeList(kindBraces, Literal("3"), Literal("4")),
					},
				},
			),
		},
	}
	runParseCase(t, data)
}

func testParseArithmetic(t *testing.T) {
	data := []ParseCase{
		{
			Input: `echo $((1 + 2 * $VAR))`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeExpr(
					makeEval(plus,
						Number(1),
						makeEval(mul, Number(2), Variable("VAR")),
					),
				),
			),
		},
		{
			Input: `echo $(((1 + 2) * $VAR))`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeExpr(
					makeEval(
						mul,
						makeEval(plus, Number(1), Number(2)),
						Variable("VAR"),
					),
				),
			),
		},
		{
			Input: `echo $((-1+1))`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeExpr(
					makeEval(plus,
						prefix{op: minus, right: Number(1)},
						Number(1),
					),
				),
			),
		},
		{
			Input: `echo $(( 1 << 2))`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeExpr(
					makeEval(tokLeftShift,
						Number(1),
						Number(2),
					),
				),
			),
		},
		{
			Input: `echo $(( 4 >> 1))`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeExpr(
					makeEval(tokRightShift,
						Number(4),
						Number(1),
					),
				),
			),
		},
	}
	runParseCase(t, data)
}

func testParseSubstitution(t *testing.T) {
	data := []ParseCase{
		{
			Input: `echo $(cat | wc)`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeList(kindSub,
					makeList(kindPipe, Literal("cat"), Literal("wc")),
				),
			),
		},
		{
			Input: `echo "sum = $(cat ; wc)"`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeList(kindSimple,
					Literal("sum = "),
					makeList(kindSub,
						makeList(kindSeq, Literal("cat"), Literal("wc")),
					),
				),
			),
		},
		{
			Input: `echo $(cat $(grep foobar) wc)`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeList(kindSub,
					makeList(kindSimple,
						Literal("cat"),
						makeList(kindSub, makeList(kindSimple,
							Literal("grep"),
							Literal("foobar"),
						)),
						Literal("wc"),
					),
				),
			),
		},
		{
			Input: `echo $(cat $(grep) $(wc))`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeList(kindSub,
					makeList(kindSimple,
						Literal("cat"),
						makeList(kindSub, makeList(kindSimple, Literal("grep"))),
						makeList(kindSub, makeList(kindSimple, Literal("wc"))),
					),
				),
			),
		},
	}
	runParseCase(t, data)
}

func testParseSimple(t *testing.T) {
	data := []ParseCase{
		{
			Input: "echo",
			Word:  makeList(kindSimple, Literal("echo")),
		},
		{
			Input: "echo foobar",
			Word:  makeList(kindSimple, Literal("echo"), Literal("foobar")),
		},
		{
			Input: "echo \"foobar\"",
			Word:  makeList(kindSimple, Literal("echo"), Literal("foobar")),
		},
		{
			Input: "echo $FOO",
			Word:  makeList(kindSimple, Literal("echo"), Variable("FOO")),
		},
		{
			Input: "echo; cat; wc; grep",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo")),
				makeList(kindSimple, Literal("cat")),
				makeList(kindSimple, Literal("wc")),
				makeList(kindSimple, Literal("grep")),
			),
		},
		{
			Input: "echo foo; echo $FOO",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo"), Literal("foo")),
				makeList(kindSimple, Literal("echo"), Variable("FOO")),
			),
		},
		{
			Input: "echo foo | echo",
			Word: makeList(kindPipe,
				makeList(kindSimple, Literal("echo"), Literal("foo")),
				makeList(kindSimple, Literal("echo")),
			),
		},
		{
			Input: "find | cat | grep",
			Word:  makeList(kindPipe, Literal("find"), Literal("cat"), Literal("grep")),
		},
		{
			Input: "find | cat | grep; wc",
			Word: makeList(kindSeq,
				makeList(kindPipe, Literal("find"), Literal("cat"), Literal("grep")),
				makeList(kindSimple, Literal("wc")),
			),
		},
		{
			Input: "echo foo && echo $FOO",
			Word: makeList(kindAnd,
				makeList(kindSimple, Literal("echo"), Literal("foo")),
				makeList(kindSimple, Literal("echo"), Variable("FOO")),
			),
		},
		{
			Input: "echo $BAR || echo $FOO",
			Word: makeList(kindOr,
				makeList(kindSimple, Literal("echo"), Variable("BAR")),
				makeList(kindSimple, Literal("echo"), Variable("FOO")),
			),
		},
		{
			Input: "echo; echo | cat",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo")),
				makeList(kindPipe,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("cat")),
				),
			),
		},
		{
			Input: "echo; echo $FOO && echo",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo")),
				makeList(kindAnd,
					makeList(kindSimple, Literal("echo"), Variable("FOO")),
					makeList(kindSimple, Literal("echo")),
				),
			),
		},
		{
			Input: "echo | cat; echo",
			Word: makeList(kindSeq,
				makeList(kindPipe,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("cat")),
				),
				makeList(kindSimple, Literal("echo")),
			),
		},
		{
			Input: "echo | cat && echo", // as (echo foo | echo) && echo bar
			Word: makeList(kindAnd,
				makeList(kindPipe,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("cat")),
				),
				makeList(kindSimple, Literal("echo")),
			),
		},
		{
			Input: "echo || echo | cat", // as echo foo || (echo bar | cat)
			Word: makeList(kindOr,
				makeList(kindSimple, Literal("echo")),
				makeList(kindPipe,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("cat")),
				),
			),
		},
		{
			Input: "echo && wc; cat && grep && sort",
			Word: makeList(kindSeq,
				makeList(kindAnd,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("wc")),
				),
				makeList(kindAnd,
					makeList(kindSimple, Literal("cat")),
					makeList(kindSimple, Literal("grep")),
					makeList(kindSimple, Literal("sort")),
				),
			),
		},
		{
			Input: "echo && wc; cat && grep || sort",
			Word: makeList(kindSeq,
				makeList(kindAnd,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("wc")),
				),
				makeList(kindOr,
					makeList(kindAnd,
						makeList(kindSimple, Literal("cat")),
						makeList(kindSimple, Literal("grep")),
					),
					makeList(kindSimple, Literal("sort")),
				),
			),
		},
		{
			Input: "echo && wc; cat || grep && sort",
			Word: makeList(kindSeq,
				makeList(kindAnd,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("wc")),
				),
				makeList(kindAnd,
					makeList(kindOr,
						makeList(kindSimple, Literal("cat")),
						makeList(kindSimple, Literal("grep")),
					),
					makeList(kindSimple, Literal("sort")),
				),
			),
		},
		{
			Input: "echo; cat || grep; sort",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo")),
				makeList(kindOr,
					makeList(kindSimple, Literal("cat")),
					makeList(kindSimple, Literal("grep")),
				),
				makeList(kindSimple, Literal("sort")),
			),
		},
	}
	runParseCase(t, data)
}

func runParseCase(t *testing.T, data []ParseCase) {
	t.Helper()

	for i, d := range data {
		w, err := Parse(d.Input)
		if err != nil {
			t.Errorf("parsing %s: unexpected error: %s", d.Input, err)
			continue
		}
		if !d.Word.Equal(w) || d.Word.String() != w.String() {
			t.Errorf("%d) %s: words mismatched!", i+1, d.Input)
			t.Logf("\twant: %s", d.Word)
			t.Logf("\tgot : %s", w)
		}
	}
}

func makeEval(op rune, left, right Evaluator) Evaluator {
	e := infix{
		left:  left,
		right: right,
		op:    op,
	}
	return e
}

func makeExpr(e Evaluator) Word {
	return makeList(kindExpr, Expr{expr: e})
}

func makeList(kind Kind, ws ...Word) Word {
	if len(ws) == 1 && !(kind == kindSub || kind == kindExpr) {
		return ws[0]
	}
	return List{
		words: ws,
		kind:  kind,
	}
}
