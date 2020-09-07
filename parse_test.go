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
			Input: "echo foo",
			Cmds:  []Command{echoFoo()},
		},
		{
			Input: "echo $VAR",
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
					cmd: makeList(Or{left: testSimple(), right: testSimple()}),
					csq: makeList(echoFoo(), echoBar()),
				},
			},
		},
		{
			Input: "if test; then echo foo\nelse if test; then echo bar\nfi",
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
					name: Token{Literal: "VAR", Type: TokLiteral},
					words: []Word{
						{tokens: []Token{{Literal: "foo", Type: TokLiteral}}},
						{tokens: []Token{{Literal: "bar", Type: TokLiteral}}},
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
					name: Token{Literal: "FOO", Type: TokLiteral},
					word: Word{
						tokens: []Token{{Literal: "foo", Type: TokLiteral}},
					},
				},
			},
		},
		{
			Input: "FOO=foo; BAR=bar",
			Cmds: []Command{
				Assign{
					name: Token{Literal: "FOO", Type: TokLiteral},
					word: Word{
						tokens: []Token{{Literal: "foo", Type: TokLiteral}},
					},
				},
				Assign{
					name: Token{Literal: "BAR", Type: TokLiteral},
					word: Word{
						tokens: []Token{{Literal: "bar", Type: TokLiteral}},
					},
				},
			},
		},
		{
			Input: "VAR=$FOOBAR",
			Cmds: []Command{
				Assign{
					name: Token{Literal: "VAR", Type: TokLiteral},
					word: Word{
						tokens: []Token{{Literal: "FOOBAR", Type: TokVariable}},
					},
				},
			},
		},
		{
			Input: "FOO=foo BAR=bar echo $FOO $BAR",
			Cmds: []Command{
				Simple{
					env: []Assign{
						{
							name: Token{Literal: "FOO", Type: TokLiteral},
							word: Word{
								tokens: []Token{{Literal: "foo", Type: TokLiteral}},
							},
						},
						{
							name: Token{Literal: "BAR", Type: TokLiteral},
							word: Word{
								tokens: []Token{{Literal: "bar", Type: TokLiteral}},
							},
						},
					},
					words: []Word{
						{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
						{tokens: []Token{{Literal: "FOO", Type: TokVariable}}},
						{tokens: []Token{{Literal: "BAR", Type: TokVariable}}},
					},
				},
			},
		},
		{
			Input: "case $VAR in\n\tfoo | bar)\n\techo foo\n\t;;\n\tfoobar)\n\techo bar\n\t;;\n\tesac",
			Cmds: []Command{
				Case{
					word: Word{
						tokens: []Token{{Literal: "VAR", Type: TokVariable}},
					},
					clauses: []Command{
						Clause{
							pattern: []Word{
								{tokens: []Token{{Literal: "foo", Type: TokLiteral}}},
								{tokens: []Token{{Literal: "bar", Type: TokLiteral}}},
							},
							body: makeList(echoFoo()),
						},
						Clause{
							pattern: []Word{
								{tokens: []Token{{Literal: "foobar", Type: TokLiteral}}},
							},
							body: makeList(echoBar()),
						},
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
	p, err := Parse(strings.NewReader(d.Input))
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
			{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
			{tokens: []Token{{Literal: "foo", Type: TokLiteral}}},
		},
	}
}

func echoBar() Command {
	return Simple{
		words: []Word{
			{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
			{tokens: []Token{{Literal: "bar", Type: TokLiteral}}},
		},
	}
}

func echoVar() Command {
	return Simple{
		words: []Word{
			{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
			{tokens: []Token{{Literal: "VAR", Type: TokVariable}}},
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
	return makeList(testSimple())
}

func testSimple() Command {
	return Simple{
		words: []Word{
			{tokens: []Token{{Literal: "test", Type: TokLiteral}}},
		},
	}
}

func makeList(cs ...Command) Command {
	return List{cmds: cs}
}
