package tish

import (
	"bytes"
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
)

const (
	tokEOS rune = -(iota + 1)
	tokEOW
	tokLit
	tokVar
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

type Scanner struct {
	buffer []byte

	char rune
	curr int
	next int

	tmp bytes.Buffer
}

func NewScanner(str string) *Scanner {
	s := &Scanner{
		buffer: []byte(str),
	}
	s.readRune()
	return s
}

func (s *Scanner) Scan() Token {
	if s.char != tokEOS {
		s.skip(isBlank)
	}

	var tok Token
	tok.Type = tokLit
	switch s.char {
	case tokEOS:
		return eosToken
	case squote:
		tok.Literal = s.scanQuotedStrong()
	case dquote:
		tok.Literal = s.scanQuotedWeak()
	case dollar:
		s.readRune()
		if !isAlpha(s.char) {
			tok.Type = tokIllegal
			break
		}
		tok.Literal = s.scanVariable()
		tok.Type = tokVar
	default:
		tok.Literal = s.scanWord()
	}
	s.readRune()

	return tok
}

func (s *Scanner) scanQuotedWeak() string {
	defer s.tmp.Reset()

	s.readRune()
	for s.char != dquote {
		if s.char == backslash {
			s.readRune()
		}
		s.tmp.WriteRune(s.char)
		s.readRune()
	}
	return s.tmp.String()
}

func (s *Scanner) scanQuotedStrong() string {
	defer s.tmp.Reset()

	s.readRune()
	for s.char != squote {
		s.tmp.WriteRune(s.char)
		s.readRune()
	}
	return s.tmp.String()
}

func (s *Scanner) scanVariable() string {
	defer s.tmp.Reset()

	for isAlpha(s.char) || s.char == underscore {
		s.tmp.WriteRune(s.char)
		s.readRune()
	}
	return s.tmp.String()
}

func (s *Scanner) scanWord() string {
	defer s.tmp.Reset()

	for !isBlank(s.char) {
		if s.char == backslash {
			s.readRune()
		}
		switch s.char {
		case squote:
			str := s.scanQuotedStrong()
			s.tmp.WriteString(str)
		case dquote:
			str := s.scanQuotedWeak()
			s.tmp.WriteString(str)
		default:
			if s.char == backslash {
				s.readRune()
			}
			s.tmp.WriteRune(s.char)
		}
		s.readRune()
	}
	return s.tmp.String()
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
