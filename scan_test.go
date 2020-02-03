package tish

import (
	"fmt"
	"testing"
)

func TestScannerScan(t *testing.T) {
	t.Run("words/simples", testWords)
	t.Run("words/variables", testVariables)
}

func testVariables(t *testing.T) {
	data := []struct {
		Input string
		Words []Token
	}{
		{
			Input: `echo $PWD`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "PWD", Type: Variable},
			},
		},
		{
			Input: `echo $PWD2`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "PWD2", Type: Variable},
			},
		},
		{
			Input: `echo $OLD_PWD`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "OLD_PWD", Type: Variable},
			},
		},
		{
			// words: WL[Word(echo)], WL[Word(foobar)], WL[Word(home = ), Variable(HOME)]
			Input: `echo foobar "home = $HOME"`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "foobar", Type: Literal},
			},
		},
		{
			// words: WL[Word(echo)], WL[Word(foo <), Variable(HOME), Word(> bar)]
			Input: `echo foo" <$HOME> "bar`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
			},
		},
	}
	for i, d := range data {
		if err := cmpTokens(d.Input, d.Words); err != nil {
			t.Errorf("%d) error: %s", i+1, err)
		}
	}
}

func testWords(t *testing.T) {
	data := []struct {
		Input string
		Words []Token
	}{
		{
			Input: `echo`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
			},
		},
		{
			Input: `echo foo bar`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "foo", Type: Literal},
				{Literal: "bar", Type: Literal},
			},
		},
		{
			Input: `echo foo\ bar`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "foo bar", Type: Literal},
			},
		},
		{
			Input: `echo fo\o`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "foo", Type: Literal},
			},
		},
		{
			Input: `echo "foobar"`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "foobar", Type: Literal},
			},
		},
		{
			Input: `echo 'foobar'`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "foobar", Type: Literal},
			},
		},
		{
			Input: `echo 'PWD=$PWD'`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "PWD=$PWD", Type: Literal},
			},
		},
		{
			Input: `echo "foo bar"`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "foo bar", Type: Literal},
			},
		},
		{
			Input: `echo "foo bar" "foo\" bar"`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "foo bar", Type: Literal},
				{Literal: "foo\" bar", Type: Literal},
			},
		},
		{
			Input: `echo prefix" between "suffix`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "prefix between suffix", Type: Literal},
			},
		},
		{
			Input: `echo prefix' between 'suffix`,
			Words: []Token{
				{Literal: "echo", Type: Literal},
				{Literal: "prefix between suffix", Type: Literal},
			},
		},
	}
	for i, d := range data {
		if err := cmpTokens(d.Input, d.Words); err != nil {
			t.Errorf("%d) error: %s", i+1, err)
		}
	}
}

func cmpTokens(str string, words []Token) error {
	s := NewScanner(str)
	for tok, j := s.Scan(), 0; tok.Type != EOS; tok = s.Scan() {
		if j >= len(words) {
			return fmt.Errorf("too many tokens generated! want %d, got %d", len(words), j+1)
		}
		if !tok.Equal(words[j]) {
			return fmt.Errorf("unexpected token! want %q, got %q", words[j].Literal, tok.Literal)
		}
		j++
	}
	return nil
}
