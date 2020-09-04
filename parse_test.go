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
			t.Errorf("%s: cmd mismatched! want %s, got %s", d.Input, d.Cmds[i], last)
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

func testCmd() Command {
	s := Simple{
		words: []Word{
			{tokens: []Token{{Literal: "test", Type: TokLiteral}}},
		},
	}
	return makeList(s)
}

func makeList(cs ...Command) Command {
  return List{cmds: cs}
}
