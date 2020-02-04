package tish

import (
	"fmt"
	"testing"
)

func TestScannerScan(t *testing.T) {
	data := []struct {
		Input string
		Words []Token
	}{
		{
			Input: `echo`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
			},
		},
		{
			Input: `echo foo bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo\ bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo bar", Type: tokWord},
			},
		},
		{
			Input: `echo fo\o`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
			},
		},
		{
			Input: `echo #comment`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "comment", Type: tokComment},
			},
		},
		{
			Input: `#comment`,
			Words: []Token{
				{Literal: "comment", Type: tokComment},
			},
		},
		{
			Input: `echo $VAR`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "VAR", Type: tokVar},
			},
		},
		{
			Input: `echo foo$VAR`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Literal: "VAR", Type: tokVar},
			},
		},
		{
			Input: `echo "foobar"`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
			},
		},
		{
			Input: `echo "foo\"bar"`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo\"bar", Type: tokWord},
			},
		},
		{
			Input: `echo "foo\bar"`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo\\bar", Type: tokWord},
			},
		},
		{
			Input: `echo 'foobar'`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
			},
		},
		{
			Input: `echo '' ""`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "", Type: tokWord},
				blank,
				{Literal: "", Type: tokWord},
			},
		},
		{
			Input: `echo foo" foobar "bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Literal: " foobar ", Type: tokWord},
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo' foobar 'bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Literal: " foobar ", Type: tokWord},
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo "$HOME"`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				blank,
				{Literal: "HOME", Type: tokVar},
			},
		},
		{
			Input: `echo foo"$HOME"bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				{Literal: "foo", Type: tokWord},
				{Literal: "HOME", Type: tokVar},
				{Literal: "bar", Type: tokWord},
			},
		},
		// {
		// 	Input: `echo foo" <$HOME> "bar`,
		// 	Words: []Token{
		// 		{Literal: "echo", Type: tokWord},
		// 		blank,
		// 		{Literal: "foo", Type: tokWord},
		// 		{Literal: " <", Type: tokWord},
		// 		{Literal: "HOME", Type: tokVar},
		// 		{Literal: "> ", Type: tokWord},
		// 		{Literal: "bar", Type: tokWord},
		// 	},
		// },
		// {
		// 	Input: `echo foobar $(echo foobar)`,
		// 	Words: []Token{
		// 		{Literal: "echo", Type: tokWord},
		// 		blank,
		// 		{Literal: "foobar", Type: tokWord},
		// 		blank,
		// 		{Type: tokBeginSub},
		// 		{Literal: "echo", Type: tokWord},
		// 		blank,
		// 		{Literal: "foobar", Type: tokWord},
		// 		{Type: tokEndSub},
		// 	},
		// },
		// {
		// 	Input: `echo foobar $(echo foobar "home = $HOME")`,
		// 	Words: []Token{
		// 		{Literal: "echo", Type: tokWord},
		// 		blank,
		// 		{Literal: "foobar", Type: tokWord},
		// 		blank,
		// 		{Type: tokBeginSub},
		// 		{Literal: "echo", Type: tokWord},
		// 		blank,
		// 		{Literal: "foobar", Type: tokWord},
		// 		blank,
		// 		{Literal: "home = ", Type: tokWord},
		// 		{Literal: "HOME", Type: tokVar},
		// 		{Type: tokEndSub},
		// 	},
		// },
		// {
		// 	Input: `echo foobar $(echo foobar $(echo foobar))`,
		// 	Words: []Token{
		// 		{Literal: "echo", Type: tokWord},
		// 		blank,
		// 		{Literal: "foobar", Type: tokWord},
		// 		blank,
		// 	}
		// },
	}
	for i, d := range data {
		if err := cmpTokens(d.Input, d.Words); err != nil {
			t.Errorf("%d) fail %s: %s", i+1, d.Input, err)
		}
	}
}

func cmpTokens(str string, words []Token) error {
	s := NewScanner(str)
	ts := make([]Token, 0, len(words))
	for tok, j := s.Scan(), 0; !tok.Equal(eof); tok = s.Scan() {
		if j >= len(words) {
			return fmt.Errorf("too many tokens generated! want %d, got %d (%s)", len(words), j+1, tok)
		}
		if !tok.Equal(words[j]) {
			return fmt.Errorf("unexpected token (%d)! want %s, got %s", j+1, words[j], tok)
		}
		ts = append(ts, tok)
		j++
	}
	return nil
}
