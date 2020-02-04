package tish

import (
	"fmt"
	"testing"
)

func TestScannerScan(t *testing.T) {
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
				eowToken,
				{Literal: "foo", Type: tokWord},
				eowToken,
				{Literal: "bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo\ bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "foo bar", Type: tokWord},
			},
		},
		{
			Input: `echo fo\o`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "foo", Type: tokWord},
			},
		},
		{
			Input: `echo #comment`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "comment", Type: tokComment},
			},
		},
		{
			Input: `#comment`,
			Words: []Token{
				{Literal: "comment", Type: tokComment},
			},
		},
		{
			Input: `echo $VAR`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "VAR", Type: tokVar},
			},
		},
		{
			Input: `echo foo$VAR`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "foo", Type: tokWord},
				{Literal: "VAR", Type: tokVar},
			},
		},
		{
			Input: `echo "foobar"`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "foobar", Type: tokWord},
			},
		},
		{
			Input: `echo "foo\"bar"`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "foo\"bar", Type: tokWord},
			},
		},
		{
			Input: `echo 'foobar'`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "foobar", Type: tokWord},
			},
		},
		{
			Input: `echo '' ""`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "", Type: tokWord},
				eowToken,
				{Literal: "", Type: tokWord},
			},
		},
		{
			Input: `echo foo" foobar "bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "foo foobar bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo' foobar 'bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "foo foobar bar", Type: tokWord},
			},
		},
		{
			Input: `echo foo" <$HOME> "bar`,
			Words: []Token{
				{Literal: "echo", Type: tokWord},
				eowToken,
				{Literal: "foo <", Type: tokWord},
				{Literal: "HOME", Type: tokVar},
				{Literal: "> bar", Type: tokWord},
			},
		},
	}
	for i, d := range data {
		if err := cmpTokens(d.Input, d.Words); err != nil {
			t.Errorf("%d) fail %s: %s", i+1, d.Input, err)
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
			return fmt.Errorf("unexpected token! want %s, got %s", words[j], tok)
		}
		j++
	}
	return nil
}
