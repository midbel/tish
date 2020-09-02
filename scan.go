package tish

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"unicode/utf8"
)

type Kind rune

const (
	TokEOF Kind = -(iota + 1)
	TokBlank
	TokLiteral
	TokVariable
	TokComment
	TokInvalid
	TokSemicolon
)

func (k Kind) String() string {
	var str string
	switch k {
	case TokEOF:
		str = "eof"
	case TokBlank:
		str = "blank"
	case TokLiteral:
		str = "literal"
	case TokVariable:
		str = "variable"
	case TokComment:
		str = "comment"
	case TokInvalid:
		str = "invalid"
	case TokSemicolon:
		str = "semicolon"
	default:
		str = "unknown"
	}
	return str
}

type Token struct {
	Literal string
	Type    Kind
}

func (t Token) Equal(other Token) bool {
	return t.Type == other.Type && t.Literal == other.Literal
}

func (t Token) String() string {
	switch t.Type {
	case TokLiteral, TokComment, TokInvalid, TokVariable:
		return fmt.Sprintf("<%s(%s)>", t.Type, t.Literal)
	default:
		return fmt.Sprintf("<%s>", t.Type)
	}
}

const (
	space      = ' '
	tab        = '\t'
	squote     = '\''
	dquote     = '"'
	newline    = '\n'
	carriage   = '\r'
	backslash  = '\\'
	semicolon  = ';'
	pound      = '#'
	dollar     = '$'
	underscore = '_'
)

type Scanner struct {
	buffer []byte
	char   rune
	curr   int
	next   int
}

func NewScanner(r io.Reader) (*Scanner, error) {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var s Scanner
	s.buffer = bytes.ReplaceAll(buf, []byte{carriage, newline}, []byte{newline})

	s.readRune()
	s.skip(isSpace)
	return &s, nil
}

func (s *Scanner) Next() Token {
	var t Token
	if s.isDone() {
		t.Type = TokEOF
		return t
	}

	switch s.char {
	case squote, dquote:
		s.scanQuote(&t)
	case dollar:
		s.scanVariable(&t)
	case pound:
		s.scanComment(&t)
	case newline, semicolon:
		s.readRune()
		s.skip(isSpace)

		t.Type = TokSemicolon
	case space, tab:
		s.scanBlank(&t)
		if t.Type != TokBlank {
			return s.Next()
		}
	default:
		s.scanLiteral(&t)
	}
	return t
}

func (s *Scanner) scanBlank(t *Token) {
	s.skip(isSpace)
	if s.isDone() || s.char == semicolon || s.char == newline || isComment(s.char) {
		return
	}
	t.Type = TokBlank
}

func (s *Scanner) scanLiteral(t *Token) {
	isDelimited := func() bool {
		return s.isDone() || isBlank(s.char) || isComment(s.char) || isQuote(s.char) || s.char == semicolon
	}

	var buf bytes.Buffer
	for {
		if s.char == backslash {
			s.readRune()
		}
		buf.WriteRune(s.char)

		s.readRune()
		if isDelimited() {
			break
		}
	}
	t.Literal = buf.String()
	t.Type = TokLiteral
}

func (s *Scanner) scanVariable(t *Token) {
	isDelimited := func() bool {
		return s.isDone() || isBlank(s.char) || isComment(s.char) || s.char == semicolon
	}

	s.readRune()
	var buf bytes.Buffer
	for {
		if !isAlpha(s.char) {
			t.Type = TokInvalid
			break
		}
		buf.WriteRune(s.char)
		s.readRune()
		if isDelimited() {
			break
		}
	}

	if t.Type != TokInvalid {
		t.Type = TokVariable
	}
	t.Literal = buf.String()
}

func (s *Scanner) scanQuote(t *Token) {
	var (
		buf   bytes.Buffer
		quote = s.char
	)
	s.readRune()
	for s.char != quote {
		if s.isDone() || s.char == newline {
			t.Type = TokInvalid
			break
		}
		if quote == dquote && s.char == backslash {
			s.readRune()
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	if t.Type != TokInvalid {
		t.Type = TokLiteral
	}
	t.Literal = buf.String()
	if isQuote(s.char) && s.char == quote {
		s.readRune()
	}
}

func (s *Scanner) scanComment(t *Token) {
	s.readRune()
	s.skip(isSpace)

	var buf bytes.Buffer
	for {
		buf.WriteRune(s.char)
		s.readRune()
		if s.isDone() || s.char == newline || s.char == pound {
			break
		}
	}
	t.Type = TokComment
	t.Literal = buf.String()
}

func (s *Scanner) skip(fn func(rune) bool) {
	for fn(s.char) {
		s.readRune()
	}
}

func (s *Scanner) readRune() {
	c, z := utf8.DecodeRune(s.buffer[s.next:])
	if c == utf8.RuneError {
		if z == 0 {
			s.curr = s.next
		}
		return
	}
	s.char, s.curr, s.next = c, s.next, s.next+z
}

// func (s *Scanner) peekRune() rune {
// 	r, _ := utf8.DecodeRune(s.buffer[s.next:])
// 	return r
// }
//
// func (s *Scanner) unreadRune() {
// 	if s.next <= 0 {
// 		return
// 	}
// 	s.next = s.curr
// 	if s.curr != 0 {
// 		s.curr -= utf8.RuneLen(s.char)
// 	}
// }

func (s *Scanner) isDone() bool {
	return s.curr >= len(s.buffer)
}

func isAlpha(r rune) bool {
	return isLetter(r) || isDigit(r) || r == underscore
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isSpace(r rune) bool {
	return r == space || r == tab
}

func isBlank(r rune) bool {
	return isSpace(r) || r == newline
}

func isQuote(r rune) bool {
	return r == dquote || r == squote
}

func isComment(r rune) bool {
	return r == pound
}
