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
				{Literal: "foo bar", Type: TokLiteral, Quoted: true},
			},
		},
		{
			Input: "echo '\"foo bar\"'",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "\"foo bar\"", Type: TokLiteral, Quoted: true},
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
				{Literal: "foo bar", Type: TokLiteral, Quoted: true},
			},
		},
		{
			Input: "echo \"foo; bar\"",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo; bar", Type: TokLiteral, Quoted: true},
			},
		},
		{
			Input: "echo \"1+1 = $((1+1)).\"",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "1+1 = ", Type: TokLiteral, Quoted: true},
				{Type: TokBegArith},
				{Literal: "1", Type: TokNumber, Quoted: true},
				{Type: TokAdd},
				{Literal: "1", Type: TokNumber, Quoted: true},
				{Type: TokEndArith},
				{Literal: ".", Type: TokLiteral, Quoted: true},
			},
		},
		{
			Input: "echo \"1+1 = $((0x1+0o1)).\"",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "1+1 = ", Type: TokLiteral, Quoted: true},
				{Type: TokBegArith},
				{Literal: "0x1", Type: TokNumber, Quoted: true},
				{Type: TokAdd},
				{Literal: "0o1", Type: TokNumber, Quoted: true},
				{Type: TokEndArith},
				{Literal: ".", Type: TokLiteral, Quoted: true},
			},
		},
		{
			Input: "echo $((== != = << >> <<= >>= += -= && || & &= | |= ~ ^ ^= <= >= < > ! ++ -- , VAR $VAR))",
			Tokens: []Token{
				createToken("echo", TokLiteral),
				blank,
				createToken("", TokBegArith),
				createToken("", TokEqual),
				createToken("", TokNotEqual),
				createToken("", TokAssign),
				createToken("", TokLeftShift),
				createToken("", TokRightShift),
				createToken("", TokLeftShiftAssign),
				createToken("", TokRightShiftAssign),
				createToken("", TokAddAssign),
				createToken("", TokSubAssign),
				createToken("", TokAnd),
				createToken("", TokOr),
				createToken("", TokBinAnd),
				createToken("", TokBinAndAssign),
				createToken("", TokBinOr),
				createToken("", TokBinOrAssign),
				createToken("", TokBinNot),
				createToken("", TokBinXor),
				createToken("", TokBinXorAssign),
				createToken("", TokLessEq),
				createToken("", TokGreatEq),
				createToken("", TokLesser),
				createToken("", TokGreater),
				createToken("", TokNot),
				createToken("", TokIncr),
				createToken("", TokDecr),
				createToken("", TokSemicolon),
				createToken("VAR", TokVariable),
				createToken("VAR", TokVariable),
				createToken("", TokEndArith),
			},
		},
		{
			Input: "echo \"\\\"foo bar\\\"\"", // echo "\"foobar\""
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "\"foo bar\"", Type: TokLiteral, Quoted: true},
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
				{Literal: " <foobar> ", Type: TokLiteral, Quoted: true},
				{Literal: "bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo foo\" <$FOO> <$BAR> \"bar",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "foo", Type: TokLiteral},
				{Literal: " <", Type: TokLiteral, Quoted: true},
				{Literal: "FOO", Type: TokVariable, Quoted: true},
				{Literal: "> <", Type: TokLiteral, Quoted: true},
				{Literal: "BAR", Type: TokVariable, Quoted: true},
				{Literal: "> ", Type: TokLiteral, Quoted: true},
				{Literal: "bar", Type: TokLiteral},
			},
		},
		{
			Input: "echo '$FOO $BAR'",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Literal: "$FOO $BAR", Type: TokLiteral, Quoted: true},
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
				{Literal: "=foo=bar=", Type: TokLiteral, Quoted: true},
			},
		},
		{
			Input: "foo=\"bar\"",
			Tokens: []Token{
				{Literal: "foo", Type: TokLiteral},
				{Type: TokAssign},
				{Literal: "bar", Type: TokLiteral, Quoted: true},
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
			Input: "echo $((1+(2*3)/((4%5)-6)))",
			Tokens: []Token{
				{Literal: "echo", Type: TokLiteral},
				blank,
				{Type: TokBegArith},
				{Literal: "1", Type: TokNumber},
				{Type: TokAdd},
				{Type: TokBegGroup},
				{Literal: "2", Type: TokNumber},
				{Type: TokMul},
				{Literal: "3", Type: TokNumber},
				{Type: TokEndGroup},
				{Type: TokDiv},
				{Type: TokBegGroup},
				{Type: TokBegGroup},
				{Literal: "4", Type: TokNumber},
				{Type: TokMod},
				{Literal: "5", Type: TokNumber},
				{Type: TokEndGroup},
				{Type: TokSub},
				{Literal: "6", Type: TokNumber},
				{Type: TokEndGroup},
				{Type: TokEndArith},
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
			Input: "FOO=${VAR:1:7} ${CMD} FOO=BAR",
			Tokens: []Token{
				{Literal: "FOO", Type: TokLiteral},
				{Type: TokAssign},
				{Type: TokBegExp},
				{Literal: "VAR", Type: TokVariable},
				{Type: TokSlice},
				{Literal: "1", Type: TokNumber},
				{Literal: "7", Type: TokNumber},
				{Type: TokEndExp},
				blank,
				{Type: TokBegExp},
				{Literal: "CMD", Type: TokVariable},
				{Type: TokEndExp},
				blank,
				{Literal: "FOO=BAR", Type: TokLiteral},
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
		{
			Input: "${FOO:1:5}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokSlice},
				{Literal: "1", Type: TokNumber},
				{Literal: "5", Type: TokNumber},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO::5}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokSlice},
				{Literal: "0", Type: TokNumber},
				{Literal: "5", Type: TokNumber},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO:1:}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokSlice},
				{Literal: "1", Type: TokNumber},
				{Literal: "0", Type: TokNumber},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO:$VAR:5}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokSlice},
				{Literal: "VAR", Type: TokVariable},
				{Literal: "5", Type: TokNumber},
				{Type: TokEndExp},
			},
		},
		{
			Input: "${FOO:1:$VAR}",
			Tokens: []Token{
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokSlice},
				{Literal: "1", Type: TokNumber},
				{Literal: "VAR", Type: TokVariable},
				{Type: TokEndExp},
			},
		},
		{
			Input: "pre{foo,bar}post",
			Tokens: []Token{
				{Literal: "pre", Type: TokLiteral},
				{Type: TokBegBrace},
				{Literal: "foo", Type: TokLiteral},
				{Type: TokSerie},
				{Literal: "bar", Type: TokLiteral},
				{Type: TokEndBrace},
				{Literal: "post", Type: TokLiteral},
			},
		},
		{
			Input: "pre{foo,bar-{one,two,three}}post",
			Tokens: []Token{
				{Literal: "pre", Type: TokLiteral},
				{Type: TokBegBrace},
				{Literal: "foo", Type: TokLiteral},
				{Type: TokSerie},
				{Literal: "bar-", Type: TokLiteral},
				{Type: TokBegBrace},
				{Literal: "one", Type: TokLiteral},
				{Type: TokSerie},
				{Literal: "two", Type: TokLiteral},
				{Type: TokSerie},
				{Literal: "three", Type: TokLiteral},
				{Type: TokEndBrace},
				{Type: TokEndBrace},
				{Literal: "post", Type: TokLiteral},
			},
		},
		{
			Input: "pre{${FOO},${BAR}}post",
			Tokens: []Token{
				{Literal: "pre", Type: TokLiteral},
				{Type: TokBegBrace},
				{Type: TokBegExp},
				{Literal: "FOO", Type: TokVariable},
				{Type: TokEndExp},
				{Type: TokSerie},
				{Type: TokBegExp},
				{Literal: "BAR", Type: TokVariable},
				{Type: TokEndExp},
				{Type: TokEndBrace},
				{Literal: "post", Type: TokLiteral},
			},
		},
		{
			Input: "pre{0..10}post",
			Tokens: []Token{
				{Literal: "pre", Type: TokLiteral},
				{Type: TokBegBrace},
				{Literal: "0", Type: TokNumber},
				{Type: TokRange},
				{Literal: "10", Type: TokNumber},
				{Type: TokEndBrace},
				{Literal: "post", Type: TokLiteral},
			},
		},
		{
			Input: "pre{0x0..0x10..0o2}post",
			Tokens: []Token{
				{Literal: "pre", Type: TokLiteral},
				{Type: TokBegBrace},
				{Literal: "0x0", Type: TokNumber},
				{Type: TokRange},
				{Literal: "0x10", Type: TokNumber},
				{Type: TokRange},
				{Literal: "0o2", Type: TokNumber},
				{Type: TokEndBrace},
				{Literal: "post", Type: TokLiteral},
			},
		},
		{
			Input: "{foo-{1..5}-bar,bar-{5..10}-foo}",
			Tokens: []Token{
				{Type: TokBegBrace},
				{Literal: "foo-", Type: TokLiteral},
				{Type: TokBegBrace},
				{Literal: "1", Type: TokNumber},
				{Type: TokRange},
				{Literal: "5", Type: TokNumber},
				{Type: TokEndBrace},
				{Literal: "-bar", Type: TokLiteral},
				{Type: TokSerie},
				{Literal: "bar-", Type: TokLiteral},
				{Type: TokBegBrace},
				{Literal: "5", Type: TokNumber},
				{Type: TokRange},
				{Literal: "10", Type: TokNumber},
				{Type: TokEndBrace},
				{Literal: "-foo", Type: TokLiteral},
				{Type: TokEndBrace},
			},
		},
	}
	testTokens(t, data)
}

func testTokens(t *testing.T, data []ScanCase) {
	t.Helper()
	for _, d := range data {
		cmpTokens(t, d)
	}
}

func cmpTokens(t *testing.T, d ScanCase) {
	t.Helper()
	s, err := NewScanner(strings.NewReader(d.Input))
	if err != nil {
		t.Errorf("%s: fail to create scanner", d.Input)
		return
	}
	for i := 0; ; i++ {
		tok := s.Scan()
		if tok.Type == TokEOF {
			if i < len(d.Tokens) {
				t.Errorf("%d) %s: not enough tokens created! want %d, got %d", i+1, d.Input, len(d.Tokens), i+1)
			}
			break
		}
		if i >= len(d.Tokens) {
			t.Errorf("%d) %s: too many tokens created! want %d, got %d - %s", i+1, d.Input, len(d.Tokens), i+1, tok)
			return
		}
		if !tok.Equal(d.Tokens[i]) {
			t.Errorf("%d) %s: tokens mismatched! want %s, got %s", i+1, d.Input, d.Tokens[i], tok)
			break
		}
	}
}
