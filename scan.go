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

	tokBeginSub
	tokEndSub
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
	case tokBeginSub:
		return "<begin sub>"
	case tokEndSub:
		return "<end sub>"
	default:
		prefix = "unknown"
	}
	return fmt.Sprintf("<%s(%s)>", prefix, t.Literal)
}

type ScanFunc func() ScanFunc

type Scanner struct {
	buffer []byte

	char rune
	curr int
	next int

	states []ScanFunc
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
	if s.char == tokEOF {
		return eof
	}

	if isBlank(s.char) {
		s.skip(isBlank)
		return blank
	}
	defer s.tmp.Reset()

	var (
		tok   Token
		scan  = s.pop()
		typof = tokWord
	)
	if scan == nil {
		switch s.char {
		case dollar:
			scan, typof = s.scanVariable, tokVar
		case pound:
			scan, typof = s.scanComment, tokComment
		case squote:
			scan = s.scanQuotedStrong
		case dquote:
			scan = s.scanOpenWeak
		default:
			scan = s.scanDefault
		}
	}

	if scan = scan(); scan != nil {
		s.push(scan)
	}
	tok.Literal, tok.Type = s.tmp.String(), typof
	return tok
}

func (s *Scanner) scanDefault() ScanFunc {
	for !isDelim(s.char) {
		switch s.char {
		case squote:
			return s.scanQuotedStrong
		case dquote:
			return s.scanOpenWeak
		case backslash:
			s.readRune()
		default:
		}
		s.tmp.WriteRune(s.char)
		s.readRune()
	}

	return nil
}

func (s *Scanner) scanOpenWeak() ScanFunc {
	if s.char == dquote {
		s.readRune()
	}
	for s.char != dquote {
		switch s.char {
		case dollar:
			s.push(s.scanCloseWeak)
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

func (s *Scanner) scanCloseWeak() ScanFunc {
	if s.char == dquote {
		return nil
	}
	return s.scanOpenWeak()
}

func (s *Scanner) scanQuotedStrong() ScanFunc {
	s.readRune()
	for s.char != squote {
		s.tmp.WriteRune(s.char)
		s.readRune()
	}

	s.readRune()
	return nil
}

func (s *Scanner) scanVariable() ScanFunc {
	s.readRune()
	for isAlpha(s.char) {
		s.tmp.WriteRune(s.char)
		s.readRune()
	}
	return nil
}

func (s *Scanner) scanComment() ScanFunc {
	s.readRune()
	for s.char != tokEOF {
		s.tmp.WriteRune(s.char)
		s.readRune()
	}

	return nil
}

func (s *Scanner) push(fn ScanFunc) {
	s.states = append(s.states, fn)
}

func (s *Scanner) pop() ScanFunc {
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
	return r == dollar || r == lparen || r == rparen
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
