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
	TokLiteral
	TokQuoted
	TokComment
	TokInvalid
	TokSemicolon
)

func (k Kind) String() string {
	var str string
	switch k {
	case TokEOF:
		str = "eof"
	case TokLiteral:
		str = "literal"
	case TokQuoted:
		str = "quoted"
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
	case TokLiteral, TokQuoted, TokComment, TokInvalid:
		return fmt.Sprintf("<%s(%s)>", t.Type, t.Literal)
	default:
		return fmt.Sprintf("<%s>", t.Type)
	}
}

const (
	space     = ' '
	tab       = '\t'
	squote    = '\''
	dquote    = '"'
	newline   = '\n'
	carriage  = '\r'
	backslash = '\\'
	semicolon = ';'
	pound     = '#'
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
	return &s, nil
}

func (s *Scanner) Next() Token {
	var t Token
	if s.isDone() {
		t.Type = TokEOF
		return t
	}
	s.skip(isSpace)
	switch s.char {
	case squote, dquote:
		s.scanQuoted(&t)
	case pound:
		s.scanComment(&t)
	case newline, semicolon:
		t.Type = TokSemicolon
	default:
		s.scanLiteral(&t)
	}
	s.readRune()
	return t
}

func (s *Scanner) scanLiteral(t *Token) {
	isDelimited := func() bool {
		return s.isDone() || isBlank(s.char) || s.char == pound || s.char == semicolon
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

	if s.char == newline || s.char == semicolon {
		s.unreadRune()
	}
}

func (s *Scanner) scanQuoted(t *Token) {
	var (
		quote  = s.char
		escape = quote == dquote
		buf    bytes.Buffer
	)

	s.readRune()

	for s.char != quote {
		if s.isDone() || s.char == newline {
			t.Type = TokInvalid
			break
		}
		if escape && s.char == backslash {
			s.readRune()
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	if t.Type != TokInvalid {
		t.Type = TokLiteral
		if quote == dquote {
			t.Type = TokQuoted
		}
	}
	t.Literal = buf.String()
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

func (s *Scanner) peekRune() rune {
	c, _ := utf8.DecodeRune(s.buffer[s.next:])
	return c
}

func (s *Scanner) unreadRune() {
	s.next = s.curr
	if s.curr != 0 {
		s.curr -= utf8.RuneLen(s.char)
	}
}

func (s *Scanner) isDone() bool {
	return s.curr >= len(s.buffer)
}

// func isAlpha(r rune) bool {
// 	return isLetter(r) || isDigit(r)
// }
//
// func isLetter(r rune) bool {
// 	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
// }
//
// func isDigit(r rune) bool {
// 	return r >= '0' && r <= '9'
// }

func isSpace(r rune) bool {
	return r == space || r == tab
}

func isBlank(r rune) bool {
	return isSpace(r) || r == newline
}
