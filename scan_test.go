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
			Input:  "",
			Tokens: []Token{},
		},
		{
			Input: "# a comment",
			Tokens: []Token{
				{Literal: "a comment", Type: TokComment},
			},
		},
		{
			Input: "echo -e foobar # a comment",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "-e", Type: TokLiteral},
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
			Input: "echo '$FOO $BAR'",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "$FOO $BAR", Type: TokLiteral},
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
		{
			Input: "\techo foo\n\t;;\techo bar\n\t;;&\techo $VAR\n\t;&",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo", Type: TokLiteral},
				{Type: TokSemicolon},
				{Type: TokBreak},
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "bar", Type: TokLiteral},
				{Type: TokSemicolon},
				{Type: TokFallthrough},
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "VAR", Type: TokVariable},
				{Type: TokSemicolon},
				{Type: TokContinue},
			},
		},
		{
			Input: "foo | bar)",
			Tokens: []Token{
				{Literal: "foo", Type: TokLiteral},
				{Type: TokPipe},
				{Literal: "bar", Type: TokLiteral},
				{Type: TokEndGroup},
			},
		},
		{
			Input: "[[ $FOO -eq foo ]]",
			Tokens: []Token{
				{Type: TokBegTest},
				{Literal: "FOO", Type: TokLiteral},
				{Literal: "-eq", Type: TokLiteral},
				{Literal: "foo", Type: TokLiteral},
				{Type: TokEndTest},
			},
		},
		{
			Input: "$((1+2))",
			Tokens: []Token{
				{Type: TokBegArith},
				{Literal: "1", Type: TokNumber},
				{Type: TokAdd},
				{Literal: "2", Type: TokNumber},
				{Type: TokEndArith},
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
		tok := s.Scan()
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
