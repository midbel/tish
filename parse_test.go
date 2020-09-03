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
						{tokens: []Token{makeLiteral("echo")}},
						{tokens: []Token{makeLiteral("foobar")}},
					},
				},
			},
		},
    {
      Input: "echo foo; echo bar",
      Cmds: []Command{
        Simple{
          words: []Word{
            {tokens: []Token{makeLiteral("echo")}},
						{tokens: []Token{makeLiteral("foo")}},
          },
        },
        Simple{
          words: []Word{
            {tokens: []Token{makeLiteral("echo")}},
						{tokens: []Token{makeLiteral("bar")}},
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
			t.Errorf("fail to parse command: %v", err)
			return
		}
    if i >= len(d.Cmds) {
      t.Errorf("too many command created! want %d, got %d", len(d.Cmds), i+1)
      return
    }
		if !last.Equal(d.Cmds[i]) {
			t.Errorf("cmd mismatched! want %s, got %s", "", "")
			return
		}
	}
}
