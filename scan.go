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
	tokEOF rune = -(iota + 1)
	tokBlank
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
	eof   = Token{Type: tokEOF}
	blank = Token{Type: tokBlank}
)

func (t Token) Equal(other Token) bool {
	return t.Type == other.Type && t.Literal == other.Literal
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	case tokEOF:
		return "<eof>"
	case tokBlank:
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

	states []stateFn
	tmp    bytes.Buffer
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

	var (
		tok   Token
		state = s.pop()
	)
	if state == nil {
		switch s.char {
		case tokEOF:
			return eof
		case space, tab:
			s.skip(isBlank)
			return blank
		case dollar:
			state = s.scanVariable
		case pound:
			state = s.scanComment
		case squote:
			state = s.scanQuotedStrong
		case dquote:
			state = s.scanQuotedWeak
		default:
			state = s.scanDefault
		}
	}
	state, tok.Literal = state(&tok), s.tmp.String()
	if state != nil {
		s.push(state)
	}
	return tok
}

func (s *Scanner) scanDefault(tok *Token) stateFn {
	tok.Type = tokWord
	for !isDelim(s.char) {
		switch s.char {
		case squote:
			return s.scanQuotedStrong
		case dquote:
			return s.scanQuotedWeak
		case backslash:
			s.readRune()
		default:
		}
		s.tmp.WriteRune(s.char)
		s.readRune()
	}

	return nil
}

func (s *Scanner) scanQuotedStrong(tok *Token) stateFn {
	tok.Type = tokWord

	s.readRune()
	for s.char != squote {
		s.tmp.WriteRune(s.char)
		s.readRune()
	}

	s.readRune()
	return nil
}

func (s *Scanner) scanQuotedWeak(tok *Token) stateFn {
	tok.Type = tokWord

	if s.char == dquote {
		s.readRune()
	}
	for s.char != dquote {
		switch s.char {
		case dollar:
			s.push(s.scanQuotedWeak)
			return s.scanVariable
		case backslash:
			if k := s.peekRune(); k == dollar || k == dquote || k == backslash {
				s.readRune()
			}
		default:
		}
		s.tmp.WriteRune(s.char)
		s.readRune()
	}
	s.readRune()
	return nil
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
	for s.char != tokEOF {
		s.tmp.WriteRune(s.char)
		s.readRune()
	}
	tok.Type = tokComment

	return nil
}

func (s *Scanner) push(fn stateFn) {
	s.states = append(s.states, fn)
}

func (s *Scanner) pop() stateFn {
	n := len(s.states)
	if n == 0 {
		return nil
	}
	fn := s.states[n-1]
	s.states = s.states[:n-1]
	return fn
}

func (s *Scanner) readRune() {
	if s.next >= len(s.buffer) {
		s.char = tokEOF
		return
	}
	r, n := utf8.DecodeRune(s.buffer[s.next:])
	if r == utf8.RuneError {
		if n == 0 {
			s.char = tokEOF
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
		return tokEOF
	}
	r, n := utf8.DecodeRune(s.buffer[s.next:])
	if r == utf8.RuneError {
		if n == 0 {
			r = tokEOF
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

func isDelim(r rune) bool {
	return isBlank(r) || isMeta(r)
}

func isMeta(r rune) bool {
	return r == dollar
}

func isBlank(r rune) bool {
	return r == space || r == tab || r == tokEOF
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
