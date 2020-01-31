package tish

import (
	"bytes"
  "unicode/utf8"
)

const (
	space     = ' '
	tab       = '\t'
	squote    = '\''
	dquote    = '"'
	backslash = '\\'
	dollar    = '$'
	newline   = '\n'
	lparen    = '('
	rparen    = ')'
	lcurly    = '{'
	rcurly    = '}'
)

const (
  EOS rune = -(iota+1)
  Word
  Illegal
)

type Token struct {
  Literal string
  Type    rune
}

type Scanner struct {
  buffer []byte

  char rune
  curr int
  next int
}

func NewScanner(str string) *Scanner {
  s := &Scanner{
    buffer: []byte(str),
  }
  s.readRune()
  return s
}

func (s *Scanner) Scan() Token {
  var tok Token
  if s.char == EOS {
    tok.Type = EOS
    return tok
  }

  s.skip(isBlank)

  var buf bytes.Buffer
  for !isBlank(s.char) {
    if s.char == backslash {
      s.readRune()
    }
    buf.WriteRune(s.char)
    s.readRune()
  }
  tok.Literal = buf.String()
  tok.Type = Word

  // s.readRune()

  return tok
}

func (s *Scanner) readRune() {
  if s.next >= len(s.buffer) {
		s.char = EOS
		return
	}
	r, n := utf8.DecodeRune(s.buffer[s.next:])
	if r == utf8.RuneError {
		if n == 0 {
			s.char = EOS
		} else {
			s.char = Illegal
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
		return EOS
	}
	r, n := utf8.DecodeRune(s.buffer[s.next:])
	if r == utf8.RuneError {
		if n == 0 {
			r = EOS
		} else {
			r = Illegal
		}
	}
	return r
}

func (s *Scanner) skip(fn func(rune)bool) {
  for fn(s.char) {
    s.readRune()
  }
}

func isBlank(r rune) bool {
  return r == space || r == tab || r == EOS
}
