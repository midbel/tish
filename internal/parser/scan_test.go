package parser_test

import (
	"strings"
	"testing"

	"github.com/midbel/tish"
)

var tokens = []struct {
	Input  string
	Tokens []rune
}{
	{
		Input:  `echo 'foobar' # a comment`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Literal, tish.Comment},
	},
	{
		Input:  `echo "$foobar" # a comment`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Quote, tish.Variable, tish.Quote, tish.Comment},
	},
	{
		Input:  `echo err 2> err.txt`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Literal, tish.RedirectErr, tish.Literal},
	},
	{
		Input:  `echo err 2>> err.txt`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Literal, tish.AppendErr, tish.Literal},
	},
	{
		Input:  `echo out1 1> out.txt`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Literal, tish.RedirectOut, tish.Literal},
	},
	{
		Input:  `echo out1 1>> out.txt`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Literal, tish.AppendOut, tish.Literal},
	},
	{
		Input:  `echo out > out.txt`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Literal, tish.RedirectOut, tish.Literal},
	},
	{
		Input:  `echo out >> out.txt`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Literal, tish.AppendOut, tish.Literal},
	},
	{
		Input:  `echo both &> both.txt`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Literal, tish.RedirectBoth, tish.Literal},
	},
	{
		Input:  `echo both &>> both.txt`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Literal, tish.AppendBoth, tish.Literal},
	},
	{
		Input:  `echo $etc/$plug/files/*`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Variable, tish.Literal, tish.Variable, tish.Literal},
	},
	{
		Input:  `echo -F'/'`,
		Tokens: []rune{tish.Literal, tish.Blank, tish.Literal, tish.Literal},
	},
	{
		Input:  `[[test]]`,
		Tokens: []rune{tish.BegTest, tish.Literal, tish.EndTest},
	},
	{
		Input:  `[[$test]]`,
		Tokens: []rune{tish.BegTest, tish.Variable, tish.EndTest},
	},
	{
		Input:  `if [[-s testdata/foobar.txt]]; then echo ok fi`,
		Tokens: []rune{tish.Keyword, tish.BegTest, tish.FileSize, tish.Literal, tish.EndTest, tish.List, tish.Keyword, tish.Literal, tish.Blank, tish.Literal, tish.Blank, tish.Keyword},
	},
}

func TestScan(t *testing.T) {
	for _, in := range tokens {
		t.Run(in.Input, func(t *testing.T) {
			scan := tish.Scan(strings.NewReader(in.Input))
			for i := 0; ; i++ {
				tok := scan.Scan()
				if tok.Type == tish.EOF {
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
