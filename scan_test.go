package tish

import (
	"fmt"
	"testing"
)

func TestScannerScan(t *testing.T) {
	t.Run("words/simples", testWords)
}

func testWords(t *testing.T) {
	data := []struct {
		Input string
		Words []Token
	}{
		{
			Input: `echo`,
			Words: []Token{
				{Literal: "echo", Type: tokLit},
			},
		},
		{
			Input: `echo foo bar`,
			Words: []Token{
				{Literal: "echo", Type: tokLit},
				{Literal: "foo", Type: tokLit},
				{Literal: "bar", Type: tokLit},
			},
		},
		{
			Input: `echo foo\ bar`,
			Words: []Token{
				{Literal: "echo", Type: tokLit},
				{Literal: "foo bar", Type: tokLit},
			},
		},
		{
			Input: `echo fo\o`,
			Words: []Token{
				{Literal: "echo", Type: tokLit},
				{Literal: "foo", Type: tokLit},
			},
		},
		{
			Input: `echo "foobar"`,
			Words: []Token{
				{Literal: "echo", Type: tokLit},
				{Literal: "foobar", Type: tokLit},
			},
		},
		{
			Input: `echo 'foobar'`,
			Words: []Token{
				{Literal: "echo", Type: tokLit},
				{Literal: "foobar", Type: tokLit},
			},
		},
		{
			Input: `echo 'PWD=$PWD'`,
			Words: []Token{
				{Literal: "echo", Type: tokLit},
				{Literal: "PWD=$PWD", Type: tokLit},
			},
		},
		{
			Input: `echo "foo bar"`,
			Words: []Token{
				{Literal: "echo", Type: tokLit},
				{Literal: "foo bar", Type: tokLit},
			},
		},
		{
			Input: `echo "foo bar" "foo\" bar"`,
			Words: []Token{
				{Literal: "echo", Type: tokLit},
				{Literal: "foo bar", Type: tokLit},
				{Literal: "foo\" bar", Type: tokLit},
			},
		},
		{
			Input: `echo prefix" between "suffix`,
			Words: []Token{
				{Literal: "echo", Type: tokLit},
				{Literal: "prefix between suffix", Type: tokLit},
			},
		},
		{
			Input: `echo prefix' between 'suffix`,
			Words: []Token{
				{Literal: "echo", Type: tokLit},
				{Literal: "prefix between suffix", Type: tokLit},
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
	for tok, j := s.Scan(), 0; !tok.Equal(eosToken); tok = s.Scan() {
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
