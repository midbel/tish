package tish

import (
	"strings"
	"testing"
)

type ScanCase struct {
	Input  string
	Tokens []Token
}

var blank = Token{Type: TokBlank}

func TestScanner(t *testing.T) {

	data := []ScanCase{
		{
			Input: "# a comment",
			Tokens: []Token{
				{Literal: "a comment", Type: TokComment},
			},
		},
		{
			Input: "echo foobar # a comment",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foobar", Type: TokLiteral},
				{Literal: "a comment", Type: TokComment},
			},
		},
		{
			Input: "echo $FOO # a comment",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "FOO", Type: TokVariable},
				{Literal: "a comment", Type: TokComment},
			},
		},
		{
			Input: "echo pound \\#  dquote \\\"",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "pound", Type: TokLiteral},
				blank,
				{Literal: "#", Type: TokLiteral},
				blank,
				{Literal: "dquote", Type: TokLiteral},
				blank,
				{Literal: "\"", Type: TokLiteral},
			},
		},
		{
			Input: "echo 'foo bar'",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo '\"foo bar\"'",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "\"foo bar\"", Type: TokLiteral},
			},
		},
		{
			Input: "echo 'foo bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo bar", Type: TokInvalid},
			},
		},
		{
			Input: "echo \"foo bar\"",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo \"foo; bar\"",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo; bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo \"\\\"foo bar\\\"\"", // echo "\"foobar\""
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "\"foo bar\"", Type: TokLiteral},
			},
		},
		{
			Input: "echo \"foo bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo bar", Type: TokInvalid},
			},
		},
		{
			Input: "echo foo; echo bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo", Type: TokLiteral},
				{Type: TokSemicolon},
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo foo\necho bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo", Type: TokLiteral},
				{Type: TokSemicolon},
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo foo\" <foobar> \"bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo", Type: TokLiteral},
				{Literal: " <foobar> ", Type: TokLiteral},
				{Literal: "bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo foo\" <$FOO> <$BAR> \"bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo", Type: TokLiteral},
				{Literal: " <", Type: TokLiteral},
				{Literal: "FOO", Type: TokVariable},
				{Literal: "> <", Type: TokLiteral},
				{Literal: "BAR", Type: TokVariable},
				{Literal: "> ", Type: TokLiteral},
				{Literal: "bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo foo && echo bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo", Type: TokLiteral},
				{Type: TokAnd},
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo foo || echo bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo", Type: TokLiteral},
				{Type: TokOr},
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo \"=foo=bar=\"",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "=foo=bar=", Type: TokLiteral},
			},
		},
		{
			Input: "foo=\"bar\"",
			Tokens: []Token{
				{Literal: "foo", Type: TokLiteral},
				{Type: TokAssign},
				{Literal: "bar", Type: TokLiteral},
			},
		},
	}
	for _, d := range data {
		testScanner(t, d)
	}
}

func testScanner(t *testing.T, d ScanCase) {
	s, err := NewScanner(strings.NewReader(d.Input))
	if err != nil {
		t.Errorf("%s: fail to create scanner", d.Input)
		return
	}
	for i := 0; ; i++ {
		tok := s.Next()
		if tok.Type == TokEOF {
			if i < len(d.Tokens) {
				t.Errorf("%s: not enough tokens created! want %d, got %d", d.Input, len(d.Tokens), i+1)
			}
			break
		}
		if i >= len(d.Tokens) {
			t.Errorf("%s: too many tokens created! want %d, got %d", d.Input, len(d.Tokens), i+1)
			return
		}
		if !tok.Equal(d.Tokens[i]) {
			t.Errorf("%s: tokens mismatched! want %s, got %s", d.Input, d.Tokens[i], tok)
			break
		}
	}
}
