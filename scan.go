package tish

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

const (
	space      = ' '
	tab        = '\t'
	squote     = '\''
	dquote     = '"'
	backslash  = '\\'
	dollar     = '$'
	newline    = '\n'
	lparen     = '('
	rparen     = ')'
	lcurly     = '{'
	rcurly     = '}'
	underscore = '_'
	pound      = '#'
)

const (
	tokEOS rune = -(iota + 1)
	tokEOW
	tokWord
	tokVar
	tokComment
	tokIllegal
)

type Token struct {
	Literal string
	Type    rune
}

var (
	eosToken = Token{Type: tokEOS}
	eowToken = Token{Type: tokEOW}
)

func (t Token) Equal(other Token) bool {
	return t.Type == other.Type && t.Literal == other.Literal
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	case tokEOS:
		return "<eof>"
	case tokEOW:
		return "<eow>"
	case tokWord:
		prefix = "word"
	case tokVar:
		prefix = "var"
	case tokComment:
		prefix = "comment"
	case tokIllegal:
		prefix = "illegal"
	default:
		prefix = "unknown"
	}
	return fmt.Sprintf("<%s(%s)>", prefix, t.Literal)
}

type stateFn func(*Token) stateFn

type Scanner struct {
	buffer []byte

	char rune
	curr int
	next int

	state stateFn
	tmp   bytes.Buffer
}

func NewScanner(str string) *Scanner {
	s := &Scanner{
		buffer: []byte(str),
	}
	s.readRune()
	s.skip(isBlank)
	return s
}

func (s *Scanner) Scan() Token {
	defer s.tmp.Reset()

	var tok Token
	if s.state == nil {
		switch s.char {
		case tokEOS:
			return eosToken
		case space, tab:
			s.skip(isBlank)
			return eowToken
		case dollar:
			s.state = s.scanVariable
		case squote:
			s.state = s.scanQuotedStrong
		case dquote:
			s.state = s.scanQuotedWeak
		case pound:
			s.state = s.scanComment
		default:
			s.state = s.scanDefault
		}
	}
	s.state, tok.Literal = s.state(&tok), s.tmp.String()
	return tok
}

func (s *Scanner) scanDefault(tok *Token) stateFn {
	for !isBlank(s.char) {
		switch s.char {
		case dollar:
			tok.Type = tokWord
			return s.scanVariable
		case squote:
			return s.scanQuotedStrong(tok)
		case dquote:
			return s.scanQuotedWeak(tok)
		case backslash:
			s.readRune()
		default:
		}
		s.tmp.WriteRune(s.char)
		s.readRune()
	}
	tok.Type = tokWord

	return nil
}

func (s *Scanner) scanQuotedWeak(tok *Token) stateFn {
	s.readRune()
	for s.char != dquote {
		switch s.char {
		case dollar:
			tok.Type = tokWord
			return s.scanVariable
		case backslash:
			s.readRune()
		default:
		}
		s.tmp.WriteRune(s.char)
		s.readRune()
	}
	s.readRune()
	return s.scanDefault(tok)
}

func (s *Scanner) scanQuotedStrong(tok *Token) stateFn {
	s.readRune()
	for s.char != squote {
		s.tmp.WriteRune(s.char)
		s.readRune()
	}
	s.readRune()
	return s.scanDefault(tok)
}

func (s *Scanner) scanVariable(tok *Token) stateFn {
	s.readRune()
	for isAlpha(s.char) {
		s.tmp.WriteRune(s.char)
		s.readRune()
	}
	tok.Type = tokVar

	return nil
}

func (s *Scanner) scanComment(tok *Token) stateFn {
	s.readRune()
	for s.char != tokEOS {
		s.tmp.WriteRune(s.char)
		s.readRune()
	}
	tok.Type = tokComment

	return nil
}

func (s *Scanner) readRune() {
	if s.next >= len(s.buffer) {
		s.char = tokEOS
		return
	}
	r, n := utf8.DecodeRune(s.buffer[s.next:])
	if r == utf8.RuneError {
		if n == 0 {
			s.char = tokEOS
		} else {
			s.char = tokIllegal
		}
		s.next = len(s.buffer)
	}
	s.char, s.curr, s.next = r, s.next, s.next+n
}

func (s *Scanner) unreadRune() {
	if s.next <= 0 || s.char == 0 {
		return
	}

	s.next, s.curr = s.curr, s.curr-utf8.RuneLen(s.char)
	s.char, _ = utf8.DecodeRune(s.buffer[s.curr:])
}

func (s *Scanner) peekRune() rune {
	if s.next >= len(s.buffer) {
		return tokEOS
	}
	r, n := utf8.DecodeRune(s.buffer[s.next:])
	if r == utf8.RuneError {
		if n == 0 {
			r = tokEOS
		} else {
			r = tokIllegal
		}
	}
	return r
}

func (s *Scanner) skip(fn func(rune) bool) {
	for fn(s.char) {
		s.readRune()
	}
}

func isBlank(r rune) bool {
	return r == space || r == tab || r == tokEOS
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAlpha(r rune) bool {
	return isLetter(r) || isDigit(r)
}
