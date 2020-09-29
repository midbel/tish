package tish

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type ParseCase struct {
	Input string
	Cmds  []Command
}

func TestParser(t *testing.T) {
	data := []ParseCase{
		{
			Input: "echo foo # a comment",
			Cmds:  []Command{echoFoo()},
		},
		{
			Input: "echo $((1+VAR))",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						createExpr(createInfix(
							createNumber(createToken("1", TokNumber)),
							createIdentifier(createToken("VAR", TokVariable)),
							TokAdd,
						)),
					},
				},
			},
		},
		{
			Input: "echo $((-1+VAR))",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						createExpr(createInfix(
							createPrefix(createNumber(createToken("1", TokNumber)), TokSub),
							createIdentifier(createToken("VAR", TokVariable)),
							TokAdd,
						)),
					},
				},
			},
		},
		{
			Input: "echo $(((1+VAR)<<8))",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						createExpr(createInfix(
							createInfix(
								createNumber(createToken("1", TokNumber)),
								createIdentifier(createToken("VAR", TokVariable)),
								TokAdd,
							),
							createNumber(createToken("8", TokNumber)),
							TokLeftShift,
						)),
					},
				},
			},
		},
		{
			Input: "echo \"1+1 = $((1+1))\"",
			Cmds: []Command{
				Simple{
					words: []Word{
						Literal{token: Token{Literal: "echo", Type: TokLiteral}},
						createList(
							createLiteral(createQuotedToken("1+1 = ", true, TokLiteral)),
							createExpr(createInfix(
								createNumber(createQuotedToken("1", true, TokNumber)),
								createNumber(createQuotedToken("1", true, TokNumber)),
								TokAdd,
							)),
						),
					},
				},
			},
		},
		{
			Input: "echo $((1+1, 1<<4, 1 && 2))",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						createExpr(
							createInfix(
								createNumber(createToken("1", TokNumber)),
								createNumber(createToken("1", TokNumber)),
								TokAdd,
							),
							createInfix(
								createNumber(createToken("1", TokNumber)),
								createNumber(createToken("4", TokNumber)),
								TokLeftShift),
							createInfix(
								createNumber(createToken("1", TokNumber)),
								createNumber(createToken("2", TokNumber)),
								TokAnd,
							),
						),
					},
				},
			},
		},
		{
			Input: "echo \"length: ${#FOO}\"",
			Cmds: []Command{
				Simple{
					words: []Word{
						Literal{token: Token{Literal: "echo", Type: TokLiteral}},
						WordList{
							words: []Word{
								Literal{token: Token{Literal: "length: ", Type: TokLiteral, Quoted: true}},
								Length{ident: Token{Literal: "FOO", Type: TokVariable, Quoted: true}},
							},
						},
					},
				},
			},
		},
		{
			Input: "# a comment\necho $VAR",
			Cmds:  []Command{echoVar()},
		},
		{
			Input: "echo foo; echo bar",
			Cmds: []Command{
				echoFoo(),
				echoBar(),
			},
		},
		{
			Input: "echo foo && echo bar",
			Cmds: []Command{
				And{
					left:  echoFoo(),
					right: echoBar(),
				},
			},
		},
		{
			Input: "if test; then echo foo; else echo bar; fi",
			Cmds: []Command{
				If{
					cmd: testCmd(),
					csq: makeList(echoFoo()),
					alt: makeList(echoBar()),
				},
			},
		},
		{
			Input: "if test || test\nthen\necho foo\necho bar\nfi",
			Cmds: []Command{
				If{
					cmd: makeList(Or{left: simpleTest(), right: simpleTest()}),
					csq: makeList(echoFoo(), echoBar()),
				},
			},
		},
		{
			Input: "if test; then echo foo\nelif test\nthen echo bar\nfi",
			Cmds: []Command{
				If{
					cmd: testCmd(),
					csq: makeList(echoFoo()),
					alt: If{
						cmd: testCmd(),
						csq: makeList(echoBar()),
					},
				},
			},
		},
		{
			Input: "for VAR in foo bar; do echo foo; done",
			Cmds: []Command{
				For{
					ident: Token{Literal: "VAR", Type: TokLiteral},
					words: []Word{
						Literal{token: Token{Literal: "foo", Type: TokLiteral}},
						Literal{token: Token{Literal: "bar", Type: TokLiteral}},
					},
					body: bodyLoop(),
				},
			},
		},
		{
			Input: "until test; do echo foo; done",
			Cmds: []Command{
				Until{
					cmd:  testCmd(),
					body: bodyLoop(),
				},
			},
		},
		{
			Input: "while test; do echo foo; done",
			Cmds: []Command{
				While{
					cmd:  testCmd(),
					body: bodyLoop(),
				},
			},
		},
		{
			Input: "while test; do break; done",
			Cmds: []Command{
				While{
					cmd:  testCmd(),
					body: breakLoop(),
				},
			},
		},
		{
			Input: "while test; do continue; done",
			Cmds: []Command{
				While{
					cmd:  testCmd(),
					body: continueLoop(),
				},
			},
		},
		{
			Input: "until test; do echo foo; if test; then echo bar; fi done",
			Cmds: []Command{
				Until{
					cmd: testCmd(),
					body: makeList(
						echoFoo(),
						If{cmd: testCmd(), csq: makeList(echoBar())},
					),
				},
			},
		},
		{
			Input: "if test; then echo foo; while test; do echo bar; done fi",
			Cmds: []Command{
				If{
					cmd: testCmd(),
					csq: makeList(
						echoFoo(),
						While{cmd: testCmd(), body: makeList(echoBar())},
					),
				},
			},
		},
		{
			Input: "FOO=foo",
			Cmds: []Command{
				Assign{
					ident: Token{Literal: "FOO", Type: TokLiteral},
					word:  Literal{token: Token{Literal: "foo", Type: TokLiteral}},
				},
			},
		},
		{
			Input: "EMPTY=",
			Cmds: []Command{
				Assign{
					ident: Token{Literal: "EMPTY", Type: TokLiteral},
					word:  Literal{},
				},
			},
		},
		{
			Input: "FOO=foo; BAR=bar",
			Cmds: []Command{
				Assign{
					ident: Token{Literal: "FOO", Type: TokLiteral},
					word:  Literal{token: Token{Literal: "foo", Type: TokLiteral}},
				},
				Assign{
					ident: Token{Literal: "BAR", Type: TokLiteral},
					word:  Literal{token: Token{Literal: "bar", Type: TokLiteral}},
				},
			},
		},
		{
			Input: "VAR=$FOOBAR",
			Cmds: []Command{
				Assign{
					ident: Token{Literal: "VAR", Type: TokLiteral},
					word:  Literal{token: Token{Literal: "FOOBAR", Type: TokVariable}},
				},
			},
		},
		{
			Input: "FOO=foo BAR=bar echo $FOO $BAR",
			Cmds: []Command{
				Simple{
					env: []Assign{
						{
							ident: Token{Literal: "FOO", Type: TokLiteral},
							word:  Literal{token: Token{Literal: "foo", Type: TokLiteral}},
						},
						{
							ident: Token{Literal: "BAR", Type: TokLiteral},
							word:  Literal{token: Token{Literal: "bar", Type: TokLiteral}},
						},
					},
					words: []Word{
						Literal{token: Token{Literal: "echo", Type: TokLiteral}},
						Literal{token: Token{Literal: "FOO", Type: TokVariable}},
						Literal{token: Token{Literal: "BAR", Type: TokVariable}},
					},
				},
			},
		},
		{
			Input: "case $VAR in\n\tfoo | bar)\n\techo foo\n\t;;\n\tfoobar)\n\techo bar\n\t;;\n\tesac",
			Cmds: []Command{
				Case{
					word: Literal{token: Token{Literal: "VAR", Type: TokVariable}},
					clauses: []Clause{
						{
							pattern: []Word{
								Literal{token: Token{Literal: "foo", Type: TokLiteral}},
								Literal{token: Token{Literal: "bar", Type: TokLiteral}},
							},
							body: makeList(echoFoo()),
							op:   Token{Type: TokBreak},
						},
						{
							pattern: []Word{
								Literal{token: Token{Literal: "foobar", Type: TokLiteral}},
							},
							body: makeList(echoBar()),
							op:   Token{Type: TokBreak},
						},
					},
				},
			},
		},
		{
			Input: "echo ${VAR}",
			Cmds: []Command{
				Simple{
					words: []Word{
						Literal{token: Token{Literal: "echo", Type: TokLiteral}},
						Literal{token: Token{Literal: "VAR", Type: TokVariable}},
					},
				},
			},
		},
		{
			Input: "echo ${#VAR}",
			Cmds: []Command{
				Simple{
					words: []Word{
						Literal{token: Token{Literal: "echo", Type: TokLiteral}},
						Length{ident: Token{Literal: "VAR", Type: TokVariable}},
					},
				},
			},
		},
		{
			Input: "${VAR}",
			Cmds: []Command{
				Simple{
					words: []Word{
						Literal{token: Token{Literal: "VAR", Type: TokVariable}},
					},
				},
			},
		},
		{
			Input: "echo ${VAR:1:7}",
			Cmds: []Command{
				Simple{
					words: []Word{
						Literal{token: Token{Literal: "echo", Type: TokLiteral}},
						Slice{
							ident:  Token{Literal: "VAR", Type: TokVariable},
							offset: Token{Literal: "1", Type: TokNumber},
							length: Token{Literal: "7", Type: TokNumber},
						},
					},
				},
			},
		},
		{
			Input: "echo ${VAR#foo}",
			Cmds: []Command{
				Simple{
					words: []Word{
						Literal{token: Token{Literal: "echo", Type: TokLiteral}},
						Trim{
							ident: Token{Literal: "VAR", Type: TokVariable},
							str:   Token{Literal: "foo", Type: TokLiteral},
							part:  Token{Type: TokTrimPrefix},
						},
					},
				},
			},
		},
		{
			Input: "echo ${VAR/foo/bar}",
			Cmds: []Command{
				Simple{
					words: []Word{
						Literal{token: Token{Literal: "echo", Type: TokLiteral}},
						Replace{
							ident: Token{Literal: "VAR", Type: TokVariable},
							src:   Token{Literal: "foo", Type: TokLiteral},
							dst:   Token{Literal: "bar", Type: TokLiteral},
							op:    Token{Type: TokReplace},
						},
					},
				},
			},
		},
		{
			Input: "echo ${VAR,,}",
			Cmds: []Command{
				Simple{
					words: []Word{
						Literal{token: Token{Literal: "echo", Type: TokLiteral}},
						Transform{
							ident: Token{Literal: "VAR", Type: TokVariable},
							op:    Token{Type: TokLowerAll},
						},
					},
				},
			},
		},
		{
			Input: "echo {foo,bar}",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						Serie{
							words: []Word{
								createLiteral(createToken("foo", TokLiteral)),
								createLiteral(createToken("bar", TokLiteral)),
							},
						},
					},
				},
			},
		},
		{
			Input: "echo {1, 100, $VAR, 1000}",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						Serie{
							words: []Word{
								createLiteral(createToken("1", TokNumber)),
								createLiteral(createToken("100", TokNumber)),
								createLiteral(createToken("VAR", TokVariable)),
								createLiteral(createToken("1000", TokNumber)),
							},
						},
					},
				},
			},
		},
		{
			Input: "echo {1..10}",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						Range{
							first: createLiteral(createToken("1", TokNumber)),
							last:  createLiteral(createToken("10", TokNumber)),
							incr:  createLiteral(createToken("1", TokNumber)),
						},
					},
				},
			},
		},
		{
			Input: "echo {1..10..2}",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						Range{
							first: createLiteral(createToken("1", TokNumber)),
							last:  createLiteral(createToken("10", TokNumber)),
							incr:  createLiteral(createToken("2", TokNumber)),
						},
					},
				},
			},
		},
		{
			Input: "echo foo-{foo,bar}-bar",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						Serie{
							prefix: createLiteral(createToken("foo-", TokLiteral)),
							suffix: createLiteral(createToken("-bar", TokLiteral)),
							words: []Word{
								createLiteral(createToken("foo", TokLiteral)),
								createLiteral(createToken("bar", TokLiteral)),
							},
						},
					},
				},
			},
		},
		{
			Input: "echo foo-{1..10}-bar",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						Range{
							prefix: createLiteral(createToken("foo-", TokLiteral)),
							suffix: createLiteral(createToken("-bar", TokLiteral)),
							first:  createLiteral(createToken("1", TokNumber)),
							last:   createLiteral(createToken("10", TokNumber)),
							incr:   createLiteral(createToken("1", TokNumber)),
						},
					},
				},
			},
		},
		{
			Input: "echo {foo-{1..10},bar-{1..10}}",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						Serie{
							words: []Word{
								Range{
									prefix: createLiteral(createToken("foo-", TokLiteral)),
									first:  createLiteral(createToken("1", TokNumber)),
									last:   createLiteral(createToken("10", TokNumber)),
									incr:   createLiteral(createToken("1", TokNumber)),
								},
								Range{
									prefix: createLiteral(createToken("bar-", TokLiteral)),
									first:  createLiteral(createToken("1", TokNumber)),
									last:   createLiteral(createToken("10", TokNumber)),
									incr:   createLiteral(createToken("1", TokNumber)),
								},
							},
						},
					},
				},
			},
		},
		{
			Input: "echo {foo-{1..10}-bar,bar-{1..10}-foo}",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						Serie{
							words: []Word{
								Range{
									prefix: createLiteral(createToken("foo-", TokLiteral)),
									suffix: createLiteral(createToken("-bar", TokLiteral)),
									first:  createLiteral(createToken("1", TokNumber)),
									last:   createLiteral(createToken("10", TokNumber)),
									incr:   createLiteral(createToken("1", TokNumber)),
								},
								Range{
									prefix: createLiteral(createToken("bar-", TokLiteral)),
									suffix: createLiteral(createToken("-foo", TokLiteral)),
									first:  createLiteral(createToken("1", TokNumber)),
									last:   createLiteral(createToken("10", TokNumber)),
									incr:   createLiteral(createToken("1", TokNumber)),
								},
							},
						},
					},
				},
			},
		},
		{
			Input: "echo {{1..10},{10..100}}",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						Serie{
							words: []Word{
								Range{
									first: createLiteral(createToken("1", TokNumber)),
									last:  createLiteral(createToken("10", TokNumber)),
									incr:  createLiteral(createToken("1", TokNumber)),
								},
								Range{
									first: createLiteral(createToken("10", TokNumber)),
									last:  createLiteral(createToken("100", TokNumber)),
									incr:  createLiteral(createToken("1", TokNumber)),
								},
							},
						},
					},
				},
			},
		},
		{
			Input: "echo {1..10}{100..1000..10}{foo,bar}",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						Range{
							first: createLiteral(createToken("1", TokNumber)),
							last:  createLiteral(createToken("10", TokNumber)),
							incr:  createLiteral(createToken("1", TokNumber)),
							suffix: Range{
								first: createLiteral(createToken("100", TokNumber)),
								last:  createLiteral(createToken("1000", TokNumber)),
								incr:  createLiteral(createToken("10", TokNumber)),
								suffix: Serie{
									words: []Word{
										createLiteral(createToken("foo", TokLiteral)),
										createLiteral(createToken("bar", TokLiteral)),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Input: "echo pre-{1..10..1}-post; echo foobar",
			Cmds: []Command{
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						Range{
							prefix: createLiteral(createToken("pre-", TokLiteral)),
							suffix: createLiteral(createToken("-post", TokLiteral)),
							first:  createLiteral(createToken("1", TokNumber)),
							last:   createLiteral(createToken("10", TokNumber)),
							incr:   createLiteral(createToken("1", TokNumber)),
						},
					},
				},
				Simple{
					words: []Word{
						createLiteral(createToken("echo", TokLiteral)),
						createLiteral(createToken("foobar", TokLiteral)),
					},
				},
			},
		},
	}
	for _, d := range data {
		testParser(t, d)
	}
}

func testParser(t *testing.T, d ParseCase) {
	p, err := NewParser(strings.NewReader(d.Input))
	if err != nil {
		t.Errorf("fail to create parser: %v", err)
		return
	}
	for i := 0; ; i++ {
		last, err := p.Parse()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Errorf("%s: fail to parse command: %v", d.Input, err)
			return
		}
		if i >= len(d.Cmds) {
			t.Errorf("%s: too many command created! want %d, got %d", d.Input, len(d.Cmds), i+1)
			return
		}
		if !last.Equal(d.Cmds[i]) {
			t.Errorf("%s: cmd mismatched!", d.Input)
			t.Errorf("- want: %s", d.Cmds[i])
			t.Errorf("- got : %s", last)
			return
		}
	}
}

func echoFoo() Command {
	return Simple{
		words: []Word{
			Literal{token: Token{Literal: "echo", Type: TokLiteral}},
			Literal{token: Token{Literal: "foo", Type: TokLiteral}},
		},
	}
}

func echoBar() Command {
	return Simple{
		words: []Word{
			Literal{token: Token{Literal: "echo", Type: TokLiteral}},
			Literal{token: Token{Literal: "bar", Type: TokLiteral}},
		},
	}
}

func echoVar() Command {
	return Simple{
		words: []Word{
			Literal{token: Token{Literal: "echo", Type: TokLiteral}},
			Literal{token: Token{Literal: "VAR", Type: TokVariable}},
		},
	}
}

func bodyLoop() Command {
	return makeList(echoFoo())
}

func breakLoop() Command {
	return makeList(Break{})
}

func continueLoop() Command {
	return makeList(Continue{})
}

func testCmd() Command {
	return makeList(simpleTest())
}

func simpleTest() Command {
	return Simple{
		words: []Word{
			Literal{token: Token{Literal: "test", Type: TokLiteral}},
		},
	}
}

func makeList(cs ...Command) Command {
	return List{cmds: cs}
}
