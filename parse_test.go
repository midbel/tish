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
			Input: "echo foobar",
			Cmds: []Command{
				Simple{
					words: []Word{
						{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
						{tokens: []Token{{Literal: "foobar", Type: TokLiteral}}},
					},
				},
			},
		},
		{
			Input: "echo $FOOBAR",
			Cmds: []Command{
				Simple{
					words: []Word{
						{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
						{tokens: []Token{{Literal: "FOOBAR", Type: TokVariable}}},
					},
				},
			},
		},
		{
			Input: "echo foo; echo bar",
			Cmds: []Command{
				Simple{
					words: []Word{
						{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
						{tokens: []Token{{Literal: "foo", Type: TokLiteral}}},
					},
				},
				Simple{
					words: []Word{
						{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
						{tokens: []Token{{Literal: "bar", Type: TokLiteral}}},
					},
				},
			},
		},
		{
			Input: "echo foo && echo bar",
			Cmds: []Command{
				And{
					left: Simple{
						words: []Word{
							{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
							{tokens: []Token{{Literal: "foo", Type: TokLiteral}}},
						},
					},
					right: Simple{
						words: []Word{
							{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
							{tokens: []Token{{Literal: "bar", Type: TokLiteral}}},
						},
					},
				},
			},
		},
		{
			Input: "if test; then echo foo; else echo bar; fi",
			Cmds: []Command{
				If{
					cmd: List{
						cmds: []Command{
							Simple{
								words: []Word{
									{tokens: []Token{{Literal: "test", Type: TokLiteral}}},
								},
							},
						},
					},
					csq: List{
						cmds: []Command{
							Simple{
								words: []Word{
									{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
									{tokens: []Token{{Literal: "foo", Type: TokLiteral}}},
								},
							},
						},
					},
					alt: List{
						cmds: []Command{
							Simple{
								words: []Word{
									{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
									{tokens: []Token{{Literal: "bar", Type: TokLiteral}}},
								},
							},
						},
					},
				},
			},
		},
		{
			Input: "until test; do echo foo; done",
			Cmds: []Command{
				Until{
					cmd: List{
						cmds: []Command{
							Simple{
								words: []Word{
									{tokens: []Token{{Literal: "test", Type: TokLiteral}}},
								},
							},
						},
					},
					body: List{
						cmds: []Command{
							Simple{
								words: []Word{
									{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
									{tokens: []Token{{Literal: "foo", Type: TokLiteral}}},
								},
							},
						},
					},
				},
			},
		},
		{
			Input: "while test; do echo foo; done",
			Cmds: []Command{
				While{
					cmd: List{
						cmds: []Command{
							Simple{
								words: []Word{
									{tokens: []Token{{Literal: "test", Type: TokLiteral}}},
								},
							},
						},
					},
					body: List{
						cmds: []Command{
							Simple{
								words: []Word{
									{tokens: []Token{{Literal: "echo", Type: TokLiteral}}},
									{tokens: []Token{{Literal: "foo", Type: TokLiteral}}},
								},
							},
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
			t.Errorf("%s: cmd mismatched! want %s, got %s", d.Input, d.Cmds[i], last)
			return
		}
	}
}
