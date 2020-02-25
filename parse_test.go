package tish

import (
	"strings"
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
	t.Run("assignments", testParseAssignments)
	t.Run("redirections", testParseRedirections)
	t.Run("parameters", testParseParameters)
	t.Run("pipes", testParsePipes)
}

func testParseParameters(t *testing.T) {
	data := []ParseCase{
		{
			Input: `echo ${VAR}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: Identity()},
			),
		},
		{
			Input: `echo ${#VAR}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: Length()},
			),
		},
		{
			Input: `echo ${VAR#prefix}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: TrimPrefix(Literal("prefix"), false)},
			),
		},
		{
			Input: `echo ${VAR##prefix}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: TrimPrefix(Literal("prefix"), true)},
			),
		},
		{
			Input: `echo ${VAR%suffix}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: TrimSuffix(Literal("suffix"), false)},
			),
		},
		{
			Input: `echo ${VAR%%sufix}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: TrimSuffix(Literal("suffix"), true)},
			),
		},
		{
			Input: `echo ${VAR/from/to}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{
					ident: "VAR",
					apply: Replace(Literal("from"), Literal("to")),
				},
			),
		},
		{
			Input: `echo ${VAR//from/to}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{
					ident: "VAR",
					apply: ReplaceAll(Literal("from"), Literal("to")),
				},
			),
		},
		{
			Input: `echo ${VAR/#from/to}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{
					ident: "VAR",
					apply: ReplacePrefix(Literal("from"), Literal("to")),
				},
			),
		},
		{
			Input: `echo ${VAR/%from/to}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{
					ident: "VAR",
					apply: ReplaceSuffix(Literal("from"), Literal("to")),
				},
			),
		},
		{
			Input: `echo ${VAR:-BAR}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: GetIfUndef(Literal("BAR"))},
			),
		},
		{
			Input: `echo ${VAR:=BAR}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: SetIfUndef(Literal("BAR"))},
			),
		},
		{
			Input: `echo ${VAR:+BAR}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: GetIfDef(Literal("BAR"))},
			),
		},
		{
			Input: `echo ${VAR:0:10}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: Substring(Number(0), Number(10))},
			),
		},
		{
			Input: `echo ${VAR:(-10):10}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: Substring(Number(-10), Number(10))},
			),
		},
		{
			Input: `echo ${VAR::10}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: Substring(Number(0), Number(10))},
			),
		},
		{
			Input: `echo ${VAR:10}`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Variable{ident: "VAR", apply: Substring(Number(10), Number(0))},
			),
		},
	}
	runParseCase(t, data)
}

func testParseRedirections(t *testing.T) {
	data := []ParseCase{
		{
			Input: `echo foo > foo.txt`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Literal("foo"),
				Redirect{
					Word: Literal("foo.txt"),
					file: fdOut,
					mode: modWrite,
				},
			),
		},
		{
			Input: `echo foo > foo.txt 2>&1`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Literal("foo"),
				Redirect{
					Word: Literal("foo.txt"),
					file: fdOut,
					mode: modWrite,
				},
				Redirect{
					file: fdOut,
					mode: modRelink,
				},
			),
		},
		{
			Input: `echo < foo.txt 2>&1 >> bar.txt`,
			Word: makeList(kindSimple,
				Literal("echo"),
				Redirect{
					Word: Literal("foo.txt"),
					file: fdIn,
					mode: modRead,
				},
				Redirect{
					file: fdOut,
					mode: modRelink,
				},
				Redirect{
					Word: Literal("bar.txt"),
					file: fdOut,
					mode: modAppend,
				},
			),
		},
	}
	runParseCase(t, data)
}

func testParseAssignments(t *testing.T) {
	data := []ParseCase{
		{
			Input: `VAR=FOOBAR`,
			Word: Assignment{
				ident: "VAR",
				word:  Literal("FOOBAR"),
			},
		},
		{
			Input: `VAR="FOO BAR"`,
			Word: Assignment{
				ident: "VAR",
				word:  Literal("FOO BAR"),
			},
		},
		{
			Input: `VAR=`,
			Word: Assignment{
				ident: "VAR",
			},
		},
		{
			Input: `VAR=$(echo foobar)`,
			Word: Assignment{
				ident: "VAR",
				word: makeList(kindSub,
					makeList(kindSimple, Literal("echo"), Literal("foobar")),
				),
			},
		},
		{
			Input: `VAR=$((1+1))`,
			Word: Assignment{
				ident: "VAR",
				word:  makeExpr(makeEval(plus, Number(1), Number(1))),
			},
		},
		{
			Input: `VAR="$((1+1)) {foo,bar}"`,
			Word: Assignment{
				ident: "VAR",
				word: makeList(kindWord,
					makeExpr(makeEval(plus, Number(1), Number(1))),
					Literal(" {foo,bar}"),
				),
			},
		},
		{
			Input: `VAR="$FOO $(echo foobar) $((1+1))"`,
			Word: Assignment{
				ident: "VAR",
				word: makeList(kindWord,
					Variable{ident: "FOO", quoted: true, apply: Identity()},
					Literal(" "),
					makeList(kindSub, makeList(kindSimple, Literal("echo"), Literal("foobar"))),
					Literal(" "),
					makeExpr(makeEval(plus, Number(1), Number(1))),
				),
			},
		},
		{
			Input: `FOO=FOO BAR=BAR echo $FOO $BAR`,
			Word: makeList(kindSimple,
				Assignment{ident: "FOO", word: Literal("FOO")},
				Assignment{ident: "BAR", word: Literal("BAR")},
				makeList(kindSimple,
					Literal("echo"),
					Variable{ident: "FOO", quoted: false, apply: Identity()},
					Variable{ident: "BAR", quoted: false, apply: Identity()},
				),
			),
		},
		{
			Input: `FOO=foobar; echo $FOO`,
			Word: makeList(kindSeq,
				Assignment{ident: "FOO", word: Literal("foobar")},
				makeList(kindSimple,
					Literal("echo"),
					Variable{ident: "FOO", quoted: false, apply: Identity()},
				),
			),
		},
	}
	runParseCase(t, data)
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
		{
			Input: "{foo,bar}",
			Word: Brace{
				word: makeList(kindBraces, Literal("foo"), Literal("bar")),
			},
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
						makeEval(mul, Number(2), Variable{ident: "VAR"}),
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
						Variable{ident: "VAR"},
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
					makeList(kindPipe,
						Pipe{Word: Literal("cat"), kind: kindPipe},
						Literal("wc"),
					),
				),
			),
		},
		{
			Input: `echo "sum = $(cat ; wc)"`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeList(kindWord,
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
		{
			Input: `$(echo foo bar)`,
			Word: makeList(kindSub,
				makeList(kindSimple, Literal("echo"), Literal("foo"), Literal("bar")),
			),
		},
	}
	runParseCase(t, data)
}

func testParsePipes(t *testing.T) {
	data := []ParseCase{
		{
			Input: "echo foo | echo",
			Word: makeList(kindPipe,
				Pipe{
					Word: makeList(kindSimple, Literal("echo"), Literal("foo")),
					kind: kindPipe,
				},
				Literal("echo"),
			),
		},
		{
			Input: "echo foo |& echo",
			Word: makeList(kindPipe,
				Pipe{
					Word: makeList(kindSimple, Literal("echo"), Literal("foo")),
					kind: kindPipeBoth,
				},
				Literal("echo"),
			),
		},
		{
			Input: "echo foo |& echo | echo", // (echo foo |& echo) | echo
			Word: makeList(kindPipe,
				Pipe{
					Word: makeList(kindSimple, Literal("echo"), Literal("foo")),
					kind: kindPipeBoth,
				},
				Pipe{
					Word: Literal("echo"),
					kind: kindPipe,
				},
				Literal("echo"),
			),
		},
		{
			Input: "echo foo | echo |& echo", // (echo | foo) |& echo
			Word: makeList(kindPipe,
				Pipe{
					Word: makeList(kindSimple, Literal("echo"), Literal("foo")),
					kind: kindPipe,
				},
				Pipe{
					Word: Literal("echo"),
					kind: kindPipeBoth,
				},
				Literal("echo"),
			),
		},
		{
			Input: "find | cat | grep",
			Word: makeList(kindPipe,
				Pipe{Word: Literal("find"), kind: kindPipe},
				Pipe{Word: Literal("cat"), kind: kindPipe},
				Literal("grep")),
		},
	}
	runParseCase(t, data)
}

func testParseSimple(t *testing.T) {
	data := []ParseCase{
		{
			Input: "echo",
			Word:  Literal("echo"),
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
			Word:  makeList(kindSimple, Literal("echo"), Variable{ident: "FOO"}),
		},
		{
			Input: "echo foobar pre-\" <$HOME> \"-post",
			Word: makeList(kindSimple,
				Literal("echo"),
				Literal("foobar"),
				makeList(kindWord,
					Literal("pre-"),
					Literal(" <"),
					Variable{ident: "HOME", quoted: true, apply: Identity()},
					Literal("> "),
					Literal("-post"),
				),
			),
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
				makeList(kindSimple, Literal("echo"), Variable{ident: "FOO"}),
			),
		},
		{
			Input: "find | cat | grep; wc",
			Word: makeList(kindSeq,
				makeList(kindPipe,
					Pipe{Word: Literal("find"), kind: kindPipe},
					Pipe{Word: Literal("cat"), kind: kindPipe},
					Literal("grep")),
				makeList(kindSimple, Literal("wc")),
			),
		},
		{
			Input: "echo foo && echo $FOO",
			Word: makeList(kindAnd,
				makeList(kindSimple, Literal("echo"), Literal("foo")),
				makeList(kindSimple, Literal("echo"), Variable{ident: "FOO"}),
			),
		},
		{
			Input: "echo $BAR || echo $FOO",
			Word: makeList(kindOr,
				makeList(kindSimple, Literal("echo"), Variable{ident: "BAR"}),
				makeList(kindSimple, Literal("echo"), Variable{ident: "FOO"}),
			),
		},
		{
			Input: "echo; echo | cat",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo")),
				makeList(kindPipe,
					Pipe{Word: Literal("echo"), kind: kindPipe},
					Literal("cat"),
				),
			),
		},
		{
			Input: "echo; echo $FOO && echo",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo")),
				makeList(kindAnd,
					makeList(kindSimple, Literal("echo"), Variable{ident: "FOO"}),
					makeList(kindSimple, Literal("echo")),
				),
			),
		},
		{
			Input: "echo | cat; echo",
			Word: makeList(kindSeq,
				makeList(kindPipe,
					Pipe{Word: Literal("echo"), kind: kindPipe},
					Literal("cat"),
				),
				makeList(kindSimple, Literal("echo")),
			),
		},
		{
			Input: "echo | cat && echo", // as (echo foo | echo) && echo bar
			Word: makeList(kindAnd,
				makeList(kindPipe,
					Pipe{Word: Literal("echo"), kind: kindPipe},
					Literal("cat"),
				),
				makeList(kindSimple, Literal("echo")),
			),
		},
		{
			Input: "echo || echo | cat", // as echo foo || (echo bar | cat)
			Word: makeList(kindOr,
				Literal("echo"),
				makeList(kindPipe,
					Pipe{Word: Literal("echo"), kind: kindPipe},
					Literal("cat"),
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
		w, err := Parse(strings.NewReader(d.Input))
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
	// if kind == kindPipe || kind == kindPipeBoth {
	// 	i := List{kind: kind, words: ws}
	// 	return Pipe{
	// 		kind: kind,
	// 		Word: i.asWord(),
	// 	}
	// }
	if len(ws) == 1 && !(kind == kindSub || kind == kindExpr) {
		return ws[0]
	}
	return List{
		words: ws,
		kind:  kind,
	}
}
