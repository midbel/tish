package parser_test

import (
	"strings"
	"testing"

	"github.com/midbel/tish/internal/parser"
	"github.com/midbel/tish/internal/token"
)

var tokens = []struct {
	Input  string
	Tokens []rune
}{
	{
		Input:  `echo 'foobar' # a comment`,
		Tokens: []rune{token.Literal, token.Blank, token.Literal, token.Comment},
	},
	{
		Input:  `echo "$foobar" # a comment`,
		Tokens: []rune{token.Literal, token.Blank, token.Quote, token.Variable, token.Quote, token.Comment},
	},
	{
		Input:  `echo err 2> err.txt`,
		Tokens: []rune{token.Literal, token.Blank, token.Literal, token.RedirectErr, token.Literal},
	},
	{
		Input:  `echo err 2>> err.txt`,
		Tokens: []rune{token.Literal, token.Blank, token.Literal, token.AppendErr, token.Literal},
	},
	{
		Input:  `echo out1 1> out.txt`,
		Tokens: []rune{token.Literal, token.Blank, token.Literal, token.RedirectOut, token.Literal},
	},
	{
		Input:  `echo out1 1>> out.txt`,
		Tokens: []rune{token.Literal, token.Blank, token.Literal, token.AppendOut, token.Literal},
	},
	{
		Input:  `echo out > out.txt`,
		Tokens: []rune{token.Literal, token.Blank, token.Literal, token.RedirectOut, token.Literal},
	},
	{
		Input:  `echo out >> out.txt`,
		Tokens: []rune{token.Literal, token.Blank, token.Literal, token.AppendOut, token.Literal},
	},
	{
		Input:  `echo both &> both.txt`,
		Tokens: []rune{token.Literal, token.Blank, token.Literal, token.RedirectBoth, token.Literal},
	},
	{
		Input:  `echo both &>> both.txt`,
		Tokens: []rune{token.Literal, token.Blank, token.Literal, token.AppendBoth, token.Literal},
	},
	{
		Input:  `echo $etc/$plug/files/*`,
		Tokens: []rune{token.Literal, token.Blank, token.Variable, token.Literal, token.Variable, token.Literal},
	},
	{
		Input:  `echo -F'/'`,
		Tokens: []rune{token.Literal, token.Blank, token.Literal, token.Literal},
	},
	{
		Input:  `[[test]]`,
		Tokens: []rune{token.BegTest, token.Literal, token.EndTest},
	},
	{
		Input:  `[[$test]]`,
		Tokens: []rune{token.BegTest, token.Variable, token.EndTest},
	},
	{
		Input:  `if [[-s testdata/foobar.txt]]; then echo ok fi`,
		Tokens: []rune{token.Keyword, token.BegTest, token.FileSize, token.Literal, token.EndTest, token.List, token.Keyword, token.Literal, token.Blank, token.Literal, token.Blank, token.Keyword},
	},
}

func TestScan(t *testing.T) {
	for _, in := range tokens {
		t.Run(in.Input, func(t *testing.T) {
			scan := parser.Scan(strings.NewReader(in.Input))
			for i := 0; ; i++ {
				tok := scan.Scan()
				if tok.Type == token.EOF {
					break
				}
				if i >= len(in.Tokens) {
					t.Errorf("too many token generated! expected %d, got %d", len(in.Tokens), i)
					break
				}
				if tok.Type != in.Tokens[i] {
					t.Errorf("token mismatched %d! %s (got %d, want %d)", i+1, tok, tok.Type, in.Tokens[i])
					break
				}
			}
		})
	}
}
