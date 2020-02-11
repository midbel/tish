package tish

import (
	"io"
	"testing"
)

type ScanCase struct {
	Input string
	Words []Token
}

func TestScannerScan(t *testing.T) {
	t.Run("simple", testScanSimple)
	t.Run("substitution", testScanSubstitution)
	t.Run("arithmetic", testScanArithmetic)
	t.Run("braces", testScanBraces)
}

func testScanSimple(t *testing.T) {
	data := []ScanCase{
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
	}
	testValidTokens(t, data)
}

func testScanSubstitution(t *testing.T) {
	data := []ScanCase{
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
	}
	testValidTokens(t, data)
}

func testScanArithmetic(t *testing.T) {
	data := []ScanCase{
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
	}
	testValidTokens(t, data)
}

func testScanBraces(t *testing.T) {
	data := []ScanCase{
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
		// {
		// 	Input: `echo {1:10}`,
		// 	Words: []Token{
		// 		{Literal: "echo", Type: tokWord},
		// 		blank,
		// 		{Type: tokBeginBrace},
		// 		{Literal: "1", Type: tokWord},
		// 		{Type: colon},
		// 		{Literal: "10", Type: tokWord},
		// 		{Type: tokEndBrace},
		// 	},
		// },
		// {
		// 	Input: `echo prolog-{a:z}-epilog`,
		// 	Words: []Token{
		// 		{Literal: "echo", Type: tokWord},
		// 		blank,
		// 		{Literal: "prolog-", Type: tokWord},
		// 		{Type: tokBeginBrace},
		// 		{Literal: "a", Type: tokWord},
		// 		{Type: colon},
		// 		{Literal: "z", Type: tokWord},
		// 		{Type: tokEndBrace},
		// 		{Literal: "-epilog", Type: tokWord},
		// 	},
		// },
		{
			Input: `echo {foo,b\ar}`,
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
		{
			Input: `echo {foo-{1,2,3},bar-{1,2,3}}`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginBrace},
				{Literal: "foo-", Type: tokWord},
				{Type: tokBeginBrace},
				{Literal: "1", Type: tokWord},
				{Type: comma},
				{Literal: "2", Type: tokWord},
				{Type: comma},
				{Literal: "3", Type: tokWord},
				{Type: tokEndBrace},
				{Type: comma},
				{Literal: "bar-", Type: tokWord},
				{Type: tokBeginBrace},
				{Literal: "1", Type: tokWord},
				{Type: comma},
				{Literal: "2", Type: tokWord},
				{Type: comma},
				{Literal: "3", Type: tokWord},
				{Type: tokEndBrace},
				{Type: tokEndBrace},
			},
		},
		{
			Input: `echo "values = {foo,bar}"`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "values = {foo,bar}", Type: tokWord},
			},
		},
	}
	testValidTokens(t, data)
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
		"echo pre-{f:b",
		"echo pre-{f:b",
		"echo pre-{foo,bar",
		"(echo foobar",
	}
	for _, str := range data {
		testInvalidTokens(t, str)
	}
}

func testInvalidTokens(t *testing.T, str string) {
	t.Helper()

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
			return
		}
	}
	t.Errorf("invalid input not detected: %s", str)
}

func testValidTokens(t *testing.T, data []ScanCase) {
	t.Helper()

	for _, d := range data {
		s := NewScanner(d.Input)
		for j := 0; ; j++ {
			tok, err := s.Scan()
			if tok.Equal(eof) {
				break
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Errorf("unexpected error: %s", err)
				break
			}
			if j >= len(d.Words) {
				t.Errorf("too many tokens generated! want %d, got %d (%s)", len(d.Words), j+1, tok)
				break
			}
			if !tok.Equal(d.Words[j]) {
				t.Errorf("unexpected token (%d)! want %s, got %s", j+1, d.Words[j], tok)
				break
			}
		}
	}
}
