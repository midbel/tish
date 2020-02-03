package tish

import (
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
				{Literal: "echo", Type: tokWord},
			},
		},
		{
			Input: `echo foo bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				{Literal: "foo", Type: tokWord},
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo\ bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				{Literal: "foo bar", Type: tokWord},
			},
		},
		{
			Input: `echo fo\o`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				{Literal: "foo", Type: tokWord},
			},
		},
	}
	for _, d := range data {
		cmpTokens(t, d.Input, d.Words)
	}
}

func cmpTokens(t *testing.T, str string, words []Token) {
	t.Helper()

	s := NewScanner(str)
	for tok, j := s.Scan(), 0; !tok.Equal(eosToken); tok = s.Scan() {
		if j >= len(words) {
			t.Errorf("too many tokens generated! want %d, got %d", len(words), j+1)
		}
		if !tok.Equal(words[j]) {
			t.Errorf("unexpected token! want %q, got %q", words[j].Literal, tok.Literal)
		}
		tok = s.Scan()
		if tok.Equal(eosToken) {
			break
		}
		if j < len(words) && !tok.Equal(eowToken) {
			t.Errorf("expected eow token, got %s", tok.Literal)
		}
		j++
	}
}
