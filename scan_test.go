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
			Input: "echo foo\\\n\t\tbar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foobar", Type: TokLiteral},
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
			Input: "echo FOO=BAR",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "FOO=BAR", Type: TokLiteral},
			},
		},
		{
			Input: "echo FOO=$BAR",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "FOO=", Type: TokLiteral},
				{Literal: "BAR", Type: TokVariable},
			},
		},
		{
			Input: "FOO=foo BAR=bar echo =$FOO=$BAR=",
			Tokens: []Token{
				{Literal: "FOO", Type: TokLiteral},
				{Type: TokAssign},
				{Literal: "foo", Type: TokLiteral},
				blank,
				{Literal: "BAR", Type: TokLiteral},
				{Type: TokAssign},
				{Literal: "bar", Type: TokLiteral},
				blank,
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "=", Type: TokLiteral},
				{Literal: "FOO", Type: TokVariable},
				{Literal: "=", Type: TokLiteral},
				{Literal: "BAR", Type: TokVariable},
				{Literal: "=", Type: TokLiteral},
			},
		},
		{
			Input: "FOO=$((1/2)) echo $FOO",
			Tokens: []Token{
				{Literal: "FOO", Type: TokLiteral},
				{Type: TokAssign},
				{Type: TokBegArith},
				{Literal: "1", Type: TokNumber},
				{Type: TokDiv},
				{Literal: "2", Type: TokNumber},
				{Type: TokEndArith},
				blank,
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "FOO", Type: TokVariable},
			},
		},
		{
			Input: "FOO=",
			Tokens: []Token{
				{Literal: "FOO", Type: TokLiteral},
				{Type: TokAssign},
			},
		},
		{
			Input: "FOO= echo $FOO",
			Tokens: []Token{
				{Literal: "FOO", Type: TokLiteral},
				{Type: TokAssign},
				blank,
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "FOO", Type: TokVariable},
			},
		},
		{
			Input: "FOO= BAR= echo $FOO",
			Tokens: []Token{
				{Literal: "FOO", Type: TokLiteral},
				{Type: TokAssign},
				blank,
				{Literal: "BAR", Type: TokLiteral},
				{Type: TokAssign},
				blank,
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "FOO", Type: TokVariable},
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
			Input: "if elif fi for do done case esac in then break continue while until",
			Tokens: []Token{
				{Literal: "if", Type: TokKeyword},
				{Literal: "elif", Type: TokKeyword},
				{Literal: "fi", Type: TokKeyword},
				{Literal: "for", Type: TokKeyword},
				{Literal: "do", Type: TokKeyword},
				{Literal: "done", Type: TokKeyword},
				{Literal: "case", Type: TokKeyword},
				{Literal: "esac", Type: TokKeyword},
				{Literal: "in", Type: TokKeyword},
				{Literal: "then", Type: TokKeyword},
				{Literal: "break", Type: TokKeyword},
				{Literal: "continue", Type: TokKeyword},
				{Literal: "while", Type: TokKeyword},
				{Literal: "until", Type: TokKeyword},
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
		{
			Input: "${FOO}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO#foobar}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokTrimPrefix},
				{Literal: "foobar", Type: TokLiteral},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO##foobar}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokTrimPrefixLong},
				{Literal: "foobar", Type: TokLiteral},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO%foobar}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokTrimSuffix},
				{Literal: "foobar", Type: TokLiteral},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO%%foobar}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokTrimSuffixLong},
				{Literal: "foobar", Type: TokLiteral},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO//foo/bar}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokReplaceAll},
				{Literal: "foo", Type: TokLiteral},
				{Literal: "bar", Type: TokLiteral},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO/foo/bar}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokReplace},
				{Literal: "foo", Type: TokLiteral},
				{Literal: "bar", Type: TokLiteral},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO/#foo/bar}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokReplacePrefix},
				{Literal: "foo", Type: TokLiteral},
				{Literal: "bar", Type: TokLiteral},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO/%foo/bar}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokReplaceSuffix},
				{Literal: "foo", Type: TokLiteral},
				{Literal: "bar", Type: TokLiteral},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${#FOO}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Type: TokLen},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO,}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokLower},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO,,}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokLowerAll},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO^}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokUpper},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO^^}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokUpperAll},
				{Type: TokEndExp},
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
			t.Errorf("%s: too many tokens created! want %d, got %d - %s", d.Input, len(d.Tokens), i+1, tok)
			return
		}
		if !tok.Equal(d.Tokens[i]) {
			t.Errorf("%s: tokens mismatched! want %s, got %s", d.Input, d.Tokens[i], tok)
			break
		}
	}
}
