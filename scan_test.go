package tish

import (
	"fmt"
	"io"
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
			Input: `echo	foo			bar`,
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
			Input: `echo ${VAR}`,
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
				blank,
				{Literal: "foo", Type: tokWord},
				{Literal: "HOME", Type: tokVar},
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo" <$HOME> "bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Literal: " <", Type: tokWord},
				{Literal: "HOME", Type: tokVar},
				{Literal: "> ", Type: tokWord},
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo; echo bar;`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Type: semicolon},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
				{Type: semicolon},
			},
		},
		{
			Input: `echo xxx | echo bar;`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "xxx", Type: tokWord},
				{Type: pipe},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
				{Type: semicolon},
			},
		},
		{
			Input: `echo foo || echo bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Type: tokOr},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo && echo bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Type: tokAnd},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo "foo" && echo 'bar'`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Type: tokAnd},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo 'foo' || echo 'bar'`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Type: tokOr},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `(echo foo; echo bar)`,
			Words: []Token{
				{Type: tokBeginList},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Type: semicolon},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
				{Type: tokEndList},
			},
		},
		{
			Input: `$(echo foobar; (echo foo; echo bar))`,
			Words: []Token{
				{Type: tokBeginSub},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
				{Type: semicolon},
				{Type: tokBeginList},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Type: semicolon},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
				{Type: tokEndList},
				{Type: tokEndSub},
			},
		},
		{
			Input: `VAR=100`,
			Words: []Token{
				{Literal: "VAR", Type: tokWord},
				{Type: equal},
				{Literal: "100", Type: tokWord},
			},
		},
		{
			Input: `VAR=$FOO # comment`,
			Words: []Token{
				{Literal: "VAR", Type: tokWord},
				{Type: equal},
				{Literal: "FOO", Type: tokVar},
				blank,
				{Literal: "comment", Type: tokComment},
			},
		},
		{
			Input: `VAR=$(echo $FOO)`,
			Words: []Token{
				{Literal: "VAR", Type: tokWord},
				{Type: equal},
				{Type: tokBeginSub},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "FOO", Type: tokVar},
				{Type: tokEndSub},
			},
		},
		{
			Input: `echo $(VAR=FOO; echo $VAR) #comment`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginSub},
				{Literal: "VAR", Type: tokWord},
				{Type: equal},
				{Literal: "FOO", Type: tokWord},
				{Type: semicolon},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "VAR", Type: tokVar},
				{Type: tokEndSub},
				blank,
				{Literal: "comment", Type: tokComment},
			},
		},
		{
			Input: `echo foo $(echo bar | echo);`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				blank,
				{Type: tokBeginSub},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
				{Type: pipe},
				{Literal: "echo", Type: tokWord},
				{Type: tokEndSub},
				{Type: semicolon},
			},
		},
		{
			Input: `echo foobar $(echo foobar)`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
				blank,
				{Type: tokBeginSub},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
				{Type: tokEndSub},
			},
		},
		{
			Input: `echo foobar $(echo foobar "home = $HOME")`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
				blank,
				{Type: tokBeginSub},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
				blank,
				{Literal: "home = ", Type: tokWord},
				{Literal: "HOME", Type: tokVar},
				{Type: tokEndSub},
			},
		},
		{
			Input: `echo foobar $(echo foobar $(echo 'foobar' "VAR = $(echo $VAR)"))`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
				blank,
				{Type: tokBeginSub},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
				blank,
				{Type: tokBeginSub},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
				blank,
				{Literal: "VAR = ", Type: tokWord},
				{Type: tokBeginSub},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "VAR", Type: tokVar},
				{Type: tokEndSub},
				{Type: tokEndSub},
				{Type: tokEndSub},
			},
		},
		{
			Input: `echo $((1 + ((2 * 3) + $VAR) / $VAR))`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginArith},
				{Literal: "1", Type: tokInt},
				{Type: plus},
				{Type: lparen},
				{Type: lparen},
				{Literal: "2", Type: tokInt},
				{Type: mul},
				{Literal: "3", Type: tokInt},
				{Type: rparen},
				{Type: plus},
				{Literal: "VAR", Type: tokVar},
				{Type: rparen},
				{Type: div},
				{Literal: "VAR", Type: tokVar},
				{Type: tokEndArith},
			},
		},
		{
			Input: `echo "sum = $((3.141592 - 2))"`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "sum = ", Type: tokWord},
				{Type: tokBeginArith},
				{Literal: "3.141592", Type: tokFloat},
				{Type: minus},
				{Literal: "2", Type: tokInt},
				{Type: tokEndArith},
			},
		},
		{
			Input: `echo $((-2 + 3))`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginArith},
				{Type: minus},
				{Literal: "2", Type: tokInt},
				{Type: plus},
				{Literal: "3", Type: tokInt},
				{Type: tokEndArith},
			},
		},
		{
			Input: `echo {1,2,3}`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginBrace},
				{Literal: "1", Type: tokWord},
				{Type: comma},
				{Literal: "2", Type: tokWord},
				{Type: comma},
				{Literal: "3", Type: tokWord},
				{Type: tokEndBrace},
			},
		},
		{
			Input: `echo {1..10}`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginBrace},
				{Literal: "1", Type: tokWord},
				{Type: tokSequence},
				{Literal: "10", Type: tokWord},
				{Type: tokEndBrace},
			},
		},
		{
			Input: `echo prolog-{a..z}-epilog`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "prolog-", Type: tokWord},
				{Type: tokBeginBrace},
				{Literal: "a", Type: tokWord},
				{Type: tokSequence},
				{Literal: "z", Type: tokWord},
				{Type: tokEndBrace},
				{Literal: "-epilog", Type: tokWord},
			},
		},
		{
			Input: `echo {foo,bar}`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginBrace},
				{Literal: "foo", Type: tokWord},
				{Type: comma},
				{Literal: "bar", Type: tokWord},
				{Type: tokEndBrace},
			},
		},
		{
			Input: `echo "foobar {foo,bar}"`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar {foo,bar}", Type: tokWord},
			},
		},
		{
			Input: `echo foobar $(echo {foo,bar})`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
				blank,
				{Type: tokBeginSub},
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginBrace},
				{Literal: "foo", Type: tokWord},
				{Type: comma},
				{Literal: "bar", Type: tokWord},
				{Type: tokEndBrace},
				{Type: tokEndSub},
			},
		},
	}
	for i, d := range data {
		if err := cmpValidTokens(d.Input, d.Words); err != nil {
			t.Errorf("%d) fail %s: %s", i+1, d.Input, err)
		}
	}
}

func TestScannerScanWithError(t *testing.T) {
	data := []string{
		"echo \"foo",
		"echo 'foo",
		"echo $(echo foobar",
		"echo $(echo $(echo foobar)",
		"echo $(echo $(echo foobar) foobar",
		"echo $((3a + 4))",
		"echo $((3 + 4)",
		"echo $((3 + 4",
		"echo pre-{foo,bar",
		"echo pre-{f..b",
		"echo pre-{f.b",
		"echo pre-{foo,bar",
		"(echo foobar",
	}
	for i, str := range data {
		if err := cmpInvalidTokens(str); err != nil {
			t.Errorf("%d) fail: %s", i+1, err)
		}
	}
}

func cmpInvalidTokens(str string) error {
	s := NewScanner(str)
	for {
		tok, err := s.Scan()
		if tok.Equal(eof) {
			break
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil
		}
	}
	return fmt.Errorf("invalid input not detected: %s", str)
}

func cmpValidTokens(str string, words []Token) error {
	s := NewScanner(str)
	for j := 0; ; j++ {
		tok, err := s.Scan()
		if tok.Equal(eof) {
			break
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if j >= len(words) {
			return fmt.Errorf("too many tokens generated! want %d, got %d (%s)", len(words), j+1, tok)
		}
		if !tok.Equal(words[j]) {
			return fmt.Errorf("unexpected token (%d)! want %s, got %s", j+1, words[j], tok)
		}
	}
	return nil
}
