package tish

import (
	"io"
	"strings"
	"testing"
)

type ScanCase struct {
	Input string
	Words []Token
}

func TestScannerQuoted(t *testing.T) {
	str := `echo $VAR "$VAR"`
	words := []Token{
		{Literal: "echo", Type: tokWord},
		blank,
		{Literal: "VAR", Type: tokWord, Quoted: false},
		blank,
		{Literal: "VAR", Type: tokWord, Quoted: true},
	}
	s := NewScanner(strings.NewReader(str))
	for j := 0; ; j++ {
		tok, err := s.Scan()
		if tok.Equal(eof) {
			break
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Errorf("%s: unexpected error: %s", str, err)
			break
		}
		if j >= len(words) {
			t.Errorf("%s: too many tokens generated! want %d, got %d (%s)", str, len(words), j+1, tok)
			break
		}
		if words[j].Quoted && !tok.Quoted {
			t.Errorf("%s: token should be quoted but is not", tok)
		}
	}
}

func TestScannerScan(t *testing.T) {
	t.Run("simple", testScanSimple)
	t.Run("substitution", testScanSubstitution)
	t.Run("arithmetic", testScanArithmetic)
	t.Run("braces", testScanBraces)
	t.Run("lines", testScanLines)
	t.Run("redirections", testScanRedirections)
	t.Run("parameters", testScanParameters)
}

func testScanParameters(t *testing.T) {
	data := []ScanCase{
		{
			Input: `echo ${FOO}`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${#FOO}`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Type: tokVarLength},
				{Literal: "FOO", Type: tokVar},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO#foobar}`, // trim prefix
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokTrimPrefix},
				{Literal: "foobar", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO#$BAR}`, // trim prefix
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokTrimPrefix},
				{Literal: "BAR", Type: tokVar},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO##foobar}`, // trim longest prefix
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokTrimPrefixLong},
				{Literal: "foobar", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO%foobar}`, // trim suffix
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokTrimSuffix},
				{Literal: "foobar", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO%%foobar}`, // trim longest suffix
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokTrimSuffixLong},
				{Literal: "foobar", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO%%$BAR}`, // trim longest suffix
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokTrimSuffixLong},
				{Literal: "BAR", Type: tokVar},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO/foo/bar}`, // substitution
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokReplace},
				{Literal: "foo", Type: tokWord},
				{Type: tokReplace},
				{Literal: "bar", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO/$SRC/$DST}`, // substitution
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokReplace},
				{Literal: "SRC", Type: tokVar},
				{Type: tokReplace},
				{Literal: "DST", Type: tokVar},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO/%foo/bar}`, // substitution
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokReplaceSuffix},
				{Literal: "foo", Type: tokWord},
				{Type: tokReplace},
				{Literal: "bar", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO/#foo/bar}`, // substitution
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokReplacePrefix},
				{Literal: "foo", Type: tokWord},
				{Type: tokReplace},
				{Literal: "bar", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO//foo/bar}`, // substitution
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokReplaceAll},
				{Literal: "foo", Type: tokWord},
				{Type: tokReplace},
				{Literal: "bar", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:-BAR}`, // BAR if FOO is not set
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokGetIfUndef},
				{Literal: "BAR", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:-$BAR}`, // BAR if FOO is not set
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokGetIfUndef},
				{Literal: "BAR", Type: tokVar},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO-BAR}`, // BAR if FOO is not set
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokGetIfUndef},
				{Literal: "BAR", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:=BAR}`, // set BAR to FOO if FOO is not set
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSetIfUndef},
				{Literal: "BAR", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:=$BAR}`, // set BAR to FOO if FOO is not set
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSetIfUndef},
				{Literal: "BAR", Type: tokVar},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO=BAR}`, // set BAR to FOO if FOO is not set
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSetIfUndef},
				{Literal: "BAR", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:+BAR}`, // get BAR if FOO is set
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokGetIfDef},
				{Literal: "BAR", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:+$BAR}`, // get BAR if FOO is set
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokGetIfDef},
				{Literal: "BAR", Type: tokVar},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO+BAR}`, // get BAR if FOO is set
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokGetIfDef},
				{Literal: "BAR", Type: tokWord},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO::2}`, // slicing
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSliceOffset},
				{Literal: "0", Type: tokInt},
				{Type: tokSliceLen},
				{Literal: "2", Type: tokInt},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:2}`, // slicing
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSliceOffset},
				{Literal: "2", Type: tokInt},
				{Type: tokSliceLen},
				{Literal: "0", Type: tokInt},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:0:2}`, // slicing
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSliceOffset},
				{Literal: "0", Type: tokInt},
				{Type: tokSliceLen},
				{Literal: "2", Type: tokInt},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:$OFF:$LEN}`, // slicing
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSliceOffset},
				{Literal: "OFF", Type: tokVar},
				{Type: tokSliceLen},
				{Literal: "LEN", Type: tokVar},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:$((1+5)):$((8%5))}`, // slicing
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSliceOffset},
				{Type: tokBeginArith},
				{Literal: "1", Type: tokInt},
				{Type: plus},
				{Literal: "5", Type: tokInt},
				{Type: tokEndArith},
				{Type: tokSliceLen},
				{Type: tokBeginArith},
				{Literal: "8", Type: tokInt},
				{Type: modulo},
				{Literal: "5", Type: tokInt},
				{Type: tokEndArith},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:(0):(2)}`, // slicing
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSliceOffset},
				{Literal: "0", Type: tokInt},
				{Type: tokSliceLen},
				{Literal: "2", Type: tokInt},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:  -7:2}`, // slicing from right
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSliceOffset},
				{Literal: "-7", Type: tokInt},
				{Type: tokSliceLen},
				{Literal: "2", Type: tokInt},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:  -7:}`, // slicing from right
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSliceOffset},
				{Literal: "-7", Type: tokInt},
				{Type: tokSliceLen},
				{Literal: "0", Type: tokInt},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:  -7:-2}`, // slicing from right
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSliceOffset},
				{Literal: "-7", Type: tokInt},
				{Type: tokSliceLen},
				{Literal: "-2", Type: tokInt},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO: -7}`, // slicing from right
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSliceOffset},
				{Literal: "-7", Type: tokInt},
				{Type: tokSliceLen},
				{Literal: "0", Type: tokInt},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO:(-7):(-2)}`, // slicing from right
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokSliceOffset},
				{Literal: "-7", Type: tokInt},
				{Type: tokSliceLen},
				{Literal: "-2", Type: tokInt},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO,}`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokLower},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO,,}`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokLowerAll},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO^}`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokUpper},
				{Type: tokEndParam},
			},
		},
		{
			Input: `echo ${FOO^^}`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginParam},
				{Literal: "FOO", Type: tokVar},
				{Type: tokUpperAll},
				{Type: tokEndParam},
			},
		},
	}
	testValidTokens(t, data)
}

func testScanRedirections(t *testing.T) {
	data := []ScanCase{
		{
			Input: `cat < foo.txt`,
			Words: []Token{
				{Literal: "cat", Type: tokWord},
				{Type: tokRedirectStdin},
				{Literal: "foo.txt", Type: tokWord},
			},
		},
		{
			Input: `cat foo.txt > bar.txt`,
			Words: []Token{
				{Literal: "cat", Type: tokWord},
				blank,
				{Literal: "foo.txt", Type: tokWord},
				{Type: tokRedirectStdout},
				{Literal: "bar.txt", Type: tokWord},
			},
		},
		{
			Input: `cat foo.txt >> bar.txt`,
			Words: []Token{
				{Literal: "cat", Type: tokWord},
				blank,
				{Literal: "foo.txt", Type: tokWord},
				{Type: tokAppendStdout},
				{Literal: "bar.txt", Type: tokWord},
			},
		},
		{
			Input: `cat foo.txt 2> foo.err`,
			Words: []Token{
				{Literal: "cat", Type: tokWord},
				blank,
				{Literal: "foo.txt", Type: tokWord},
				{Type: tokRedirectStderr},
				{Literal: "foo.err", Type: tokWord},
			},
		},
		{
			Input: `cat foo.txt 2>> foo.err`,
			Words: []Token{
				{Literal: "cat", Type: tokWord},
				blank,
				{Literal: "foo.txt", Type: tokWord},
				{Type: tokAppendStderr},
				{Literal: "foo.err", Type: tokWord},
			},
		},
		{
			Input: `cat foo.txt 2>&1`,
			Words: []Token{
				{Literal: "cat", Type: tokWord},
				blank,
				{Literal: "foo.txt", Type: tokWord},
				{Type: tokRedirectErrToOut},
			},
		},
		{
			Input: `cat foo.txt >&2`,
			Words: []Token{
				{Literal: "cat", Type: tokWord},
				blank,
				{Literal: "foo.txt", Type: tokWord},
				{Type: tokRedirectOutToErr},
			},
		},
		{
			Input: `cat foo.txt &> /dev/null`,
			Words: []Token{
				{Literal: "cat", Type: tokWord},
				blank,
				{Literal: "foo.txt", Type: tokWord},
				{Type: tokRedirectBoth},
				{Literal: "/dev/null", Type: tokWord},
			},
		},
		{
			Input: `cat foo.txt &>> /dev/null`,
			Words: []Token{
				{Literal: "cat", Type: tokWord},
				blank,
				{Literal: "foo.txt", Type: tokWord},
				{Type: tokAppendBoth},
				{Literal: "/dev/null", Type: tokWord},
			},
		},
		{
			Input: `echo < foo.txt 2>&1 >> bar.txt`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				{Type: tokRedirectStdin},
				{Literal: "foo.txt", Type: tokWord},
				{Type: tokRedirectErrToOut},
				{Type: tokAppendStdout},
				{Literal: "bar.txt", Type: tokWord},
			},
		},
	}
	testValidTokens(t, data)
}

func testScanLines(t *testing.T) {
	data := []ScanCase{
		{
			Input: `
echo foo
echo bar
echo \
	foobar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Type: semicolon},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
				{Type: semicolon},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
				{Type: semicolon},
			},
		},
		{
			Input: `
echo foobar

echo foo

echo bar
			`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord},
				{Type: semicolon},
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
			Input: `
# prolog
# comment
echo foo # comment
echo bar # comment

# epilog
# comment
			`,
			Words: []Token{
				{Literal: "prolog", Type: tokComment},
				{Type: semicolon},
				{Literal: "comment", Type: tokComment},
				{Type: semicolon},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Literal: "comment", Type: tokComment},
				{Type: semicolon},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
				{Literal: "comment", Type: tokComment},
				{Type: semicolon},
				{Literal: "epilog", Type: tokComment},
				{Type: semicolon},
				{Literal: "comment", Type: tokComment},
				{Type: semicolon},
			},
		},
		{
			Input: `
echo foo
:' comment 1
comment 2
comment 3
'
echo bar
			`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Type: semicolon},
				{Literal: "comment 1\ncomment 2\ncomment 3\n", Type: tokComment},
				{Type: semicolon},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
				{Type: semicolon},
			},
		},
	}
	testValidTokens(t, data)
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
				{Literal: "comment", Type: tokComment},
				{Type: semicolon},
			},
		},
		{
			Input: `#comment`,
			Words: []Token{
				{Literal: "comment", Type: tokComment},
				{Type: semicolon},
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
				{Literal: "foobar", Type: tokWord, Quoted: true},
			},
		},
		{
			Input: `echo "foo\"bar"`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo\"bar", Type: tokWord, Quoted: true},
			},
		},
		{
			Input: `echo "foo\bar"`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo\\bar", Type: tokWord, Quoted: true},
			},
		},
		{
			Input: `echo 'foobar'`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foobar", Type: tokWord, Quoted: true},
			},
		},
		{
			Input: `echo '' ""`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "", Type: tokWord, Quoted: true},
				blank,
				{Literal: "", Type: tokWord, Quoted: true},
			},
		},
		{
			Input: `echo foo" foobar "bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Literal: " foobar ", Type: tokWord, Quoted: true},
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo' foobar 'bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Literal: " foobar ", Type: tokWord, Quoted: true},
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
				{Literal: "HOME", Type: tokVar, Quoted: true},
			},
		},
		{
			Input: `echo foo"$HOME"bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Literal: "HOME", Type: tokVar, Quoted: true},
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo" <$HOME> "bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Literal: " <", Type: tokWord, Quoted: true},
				{Literal: "HOME", Type: tokVar, Quoted: true},
				{Literal: "> ", Type: tokWord, Quoted: true},
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
			Input: `echo foo | echo bar;`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Type: tokPipe},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord},
				{Type: semicolon},
			},
		},
		{
			Input: `echo foo |& echo bar;`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord},
				{Type: tokPipeBoth},
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
				{Literal: "foo", Type: tokWord, Quoted: true},
				{Type: tokAnd},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord, Quoted: true},
			},
		},
		{
			Input: `echo 'foo' || echo 'bar'`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "foo", Type: tokWord, Quoted: true},
				{Type: tokOr},
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "bar", Type: tokWord, Quoted: true},
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
				{Literal: "comment", Type: tokComment},
				{Type: semicolon},
			},
		},
		{
			Input: `echo $$ $? $# $! $@ $$`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Literal: "$", Type: tokVar},
				blank,
				{Literal: "?", Type: tokVar},
				blank,
				{Literal: "#", Type: tokVar},
				blank,
				{Literal: "!", Type: tokVar},
				blank,
				{Literal: "@", Type: tokVar},
				blank,
				{Literal: "$", Type: tokVar},
			},
		},
		{
			Input: "cat;\ngrep;",
			Words: []Token{
				{Literal: "cat", Type: tokWord},
				{Type: semicolon},
				{Literal: "grep", Type: tokWord},
				{Type: semicolon},
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
				{Literal: "comment", Type: tokComment},
				{Type: semicolon},
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
				{Type: tokPipe},
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
				{Literal: "home = ", Type: tokWord, Quoted: true},
				{Literal: "HOME", Type: tokVar, Quoted: true},
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
				{Literal: "foobar", Type: tokWord, Quoted: true},
				blank,
				{Literal: "VAR = ", Type: tokWord, Quoted: true},
				{Type: tokBeginSub, Quoted: true},
				{Literal: "echo", Type: tokWord, Quoted: true},
				blank,
				{Literal: "VAR", Type: tokVar, Quoted: true},
				{Type: tokEndSub, Quoted: true},
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
				{Literal: "sum = ", Type: tokWord, Quoted: true},
				{Type: tokBeginArith, Quoted: true},
				{Literal: "3.141592", Type: tokFloat, Quoted: true},
				{Type: minus, Quoted: true},
				{Literal: "2", Type: tokInt, Quoted: true},
				{Type: tokEndArith, Quoted: true},
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
			Input: `echo $(((2 | 4) >> 1 ))`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginArith},
				{Type: lparen},
				{Literal: "2", Type: tokInt},
				{Type: pipe},
				{Literal: "4", Type: tokInt},
				{Type: rparen},
				{Type: tokRightShift},
				{Literal: "1", Type: tokInt},
				{Type: tokEndArith},
			},
		},
		{
			Input: `echo $((2<<-1))`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginArith},
				{Literal: "2", Type: tokInt},
				{Type: tokLeftShift},
				{Type: minus},
				{Literal: "1", Type: tokInt},
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
				{Literal: "foobar {foo,bar}", Type: tokWord, Quoted: true},
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
			Input: `echo {foo-{1,2}, bar-{3,4}}`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				blank,
				{Type: tokBeginBrace},
				{Literal: "foo-", Type: tokWord},
				{Type: tokBeginBrace},
				{Literal: "1", Type: tokWord},
				{Type: comma},
				{Literal: "2", Type: tokWord},
				{Type: tokEndBrace},
				{Type: comma},
				{Literal: "bar-", Type: tokWord},
				{Type: tokBeginBrace},
				{Literal: "3", Type: tokWord},
				{Type: comma},
				{Literal: "4", Type: tokWord},
				{Type: tokEndBrace},
				{Type: tokEndBrace},
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

	s := NewScanner(strings.NewReader(str))
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
		s := NewScanner(strings.NewReader(d.Input))
		for j := 0; ; j++ {
			tok, err := s.Scan()
			if tok.Equal(eof) {
				break
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Errorf("%s: unexpected error: %s", d.Input, err)
				break
			}
			if j >= len(d.Words) {
				t.Errorf("%s: too many tokens generated! want %d, got %d (%s)", d.Input, len(d.Words), j+1, tok)
				break
			}
			if !tok.Equal(d.Words[j]) {
				t.Errorf("%s: unexpected token (%d)! want %s, got %s", d.Input, j+1, d.Words[j], tok)
			}
		}
	}
}
