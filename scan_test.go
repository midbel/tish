package tish

import (
	"strings"
	"testing"
)

type ScanCase struct {
	Input  string
	Tokens []Token
}

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
				{Literal: "foobar", Type: TokLiteral},
				{Literal: "a comment", Type: TokComment},
			},
		},
		{
			Input: "echo pound \\#  dquote \\\"",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				{Literal: "pound", Type: TokLiteral},
				{Literal: "#", Type: TokLiteral},
				{Literal: "dquote", Type: TokLiteral},
				{Literal: "\"", Type: TokLiteral},
			},
		},
		{
			Input: "echo 'foo bar'",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				{Literal: "foo bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo '\"foo bar\"'",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				{Literal: "\"foo bar\"", Type: TokLiteral},
			},
		},
		{
			Input: "echo 'foo bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				{Literal: "foo bar", Type: TokInvalid},
			},
		},
		{
			Input: "echo \"foo bar\"",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				{Literal: "foo bar", Type: TokQuoted},
			},
		},
		{
			Input: "echo \"\\\"foo bar\\\"\"",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				{Literal: "\"foo bar\"", Type: TokQuoted},
			},
		},
		{
			Input: "echo \"foo bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				{Literal: "foo bar", Type: TokInvalid},
			},
		},
		{
			Input: "echo foo; echo bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				{Literal: "foo", Type: TokLiteral},
				{Type: TokSemicolon},
				{Literal: "echo", Type: TokLiteral},
				{Literal: "bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo foo\necho bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				{Literal: "foo", Type: TokLiteral},
				{Type: TokSemicolon},
				{Literal: "echo", Type: TokLiteral},
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
		t.Errorf("fail to create scanner: %s", d.Input)
		return
	}
	for i := 0; ; i++ {
		tok := s.Next()
		if tok.Type == TokEOF {
			if i < len(d.Tokens) {
				t.Errorf("not enough tokens created! want %d, got %d", len(d.Tokens), i+1)
			}
			break
		}
		if i >= len(d.Tokens) {
			t.Errorf("too many tokens created! want %d, got %d", len(d.Tokens), i+1)
			return
		}
		if !tok.Equal(d.Tokens[i]) {
			t.Errorf("tokens mismatched! want %s, got %s", d.Tokens[i], tok)
			break
		}
	}
}
