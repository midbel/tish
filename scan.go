package tish

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"unicode/utf8"
)

func init() {
	sort.Strings(keywords)
}

const (
	kwIf       = "if"
	kwFi       = "fi"
	kwFor      = "for"
	kwUntil    = "until"
	kwWhile    = "while"
	kwDo       = "do"
	kwDone     = "done"
	kwCase     = "case"
	kwEsac     = "esac"
	kwBreak    = "break"
	kwContinue = "continue"
	kwThen     = "then"
	kwElse     = "else"
)

var keywords = []string{
	kwIf,
	kwFi,
	kwFor,
	kwWhile,
	kwUntil,
	kwDo,
	kwDone,
	kwCase,
	kwEsac,
	kwBreak,
	kwContinue,
	kwThen,
	kwElse,
}

type Kind rune

const (
	TokEOF Kind = -(iota + 1)
	TokBlank
	TokKeyword
	TokLiteral
	TokVariable
	TokComment
	TokInvalid
	TokSemicolon
	TokAnd
	TokOr
	TokPipe
	TokBackground
	TokAssign
	TokEqual
	TokNotEqual
)

func (k Kind) EndOfWord() bool {
	return k == TokBlank || k == TokAnd || k == TokOr || k == TokSemicolon || k == TokPipe || k == TokBackground
}

func (k Kind) EndOfCommand() bool {
	return k == TokSemicolon || k == TokEOF
}

func (k Kind) String() string {
	var str string
	switch k {
	case TokEOF:
		str = "eof"
	case TokBlank:
		str = "blank"
	case TokLiteral:
		str = "literal"
	case TokKeyword:
		str = "keyword"
	case TokVariable:
		str = "variable"
	case TokComment:
		str = "comment"
	case TokInvalid:
		str = "invalid"
	case TokSemicolon:
		str = "semicolon"
	case TokAnd:
		str = "and"
	case TokOr:
		str = "or"
	case TokPipe:
		str = "pipe"
	case TokBackground:
		str = "background"
	default:
		str = "unknown"
	}
	return str
}

type Token struct {
	Literal string
	Type    Kind
	Quoted  bool
}

func (t Token) Equal(other Token) bool {
	return t.Type == other.Type && t.Literal == other.Literal && t.Quoted == other.Quoted
}

func (t Token) String() string {
	switch t.Type {
	case TokLiteral, TokComment, TokInvalid, TokVariable, TokKeyword:
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
	ampersand  = '&'
	pipe       = '|'
	plus       = '+'
	minus      = '-'
	star       = '*'
	slash      = '/'
	equal      = '='
	tilde      = '~'
	rangle     = '>'
	langle     = '<'
	lparen     = '('
	rparen     = ')'
	lsquare    = '['
	rsquare    = ']'
	lcurly     = '{'
	rcurly     = '}'
	arobase    = '@'
	percent    = '%'
	colon      = ':'
	dot        = '.'
	bang       = '!'
	question   = '?'
)

type Scanner struct {
	buffer []byte
	char   rune
	curr   int
	next   int

	split func(rune) bool
	quote rune
}

func NewScanner(r io.Reader) (*Scanner, error) {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	s := Scanner{
		buffer: bytes.ReplaceAll(buf, []byte{carriage, newline}, []byte{newline}),
		split:  splitLiteral,
	}

	s.readRune()
	s.skip(isSpace)
	return &s, nil
}

func (s *Scanner) Next() Token {
	s.switchSplit()
	var t Token
	if s.isDone() {
		t.Type = TokEOF
		return t
	}
	switch {
	case isVar(s.char):
		s.scanVariable(&t)
	case isComment(s.char):
		s.scanComment(&t)
	case isOperator(s.char):
		s.scanOperator(&t)
	case s.char == newline || s.char == semicolon:
		s.readRune()
		s.skip(isSpace)

		t.Type = TokSemicolon
	case !s.isQuoted() && isSpace(s.char):
		s.scanBlank(&t)
		if t.Type != TokBlank {
			return s.Next()
		}
	default:
		s.scanDefault(&t)
	}
	// t.Quoted = s.isQuoted()
	return t
}

func (s *Scanner) scanDefault(t *Token) {
	var buf bytes.Buffer
	for !s.isDone() && !s.split(s.char) {
		if canEscape(s.quote, s.char) {
			s.readRune()
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	t.Literal = buf.String()
	t.Type = TokLiteral
	if s.isQuoted() && (s.isDone() || s.char == newline) {
		t.Type = TokInvalid
		return
	}
	if !s.isQuoted() {
		x := sort.SearchStrings(keywords, t.Literal)
		if x < len(keywords) && keywords[x] == t.Literal {
			t.Type = TokKeyword
			s.skip(isSpace)
		}
	}
}

func (s *Scanner) scanOperator(t *Token) {
	switch s.char {
	case ampersand:
		t.Type = TokBackground
		s.readRune()
		if s.char == ampersand {
			s.readRune()
			t.Type = TokAnd
		}
		s.skip(isSpace)
	case pipe:
		t.Type = TokPipe
		s.readRune()
		if s.char == pipe {
			s.readRune()
			t.Type = TokOr
		}
		s.skip(isSpace)
	default:
		t.Type = TokInvalid
	}
}

func (s *Scanner) scanBlank(t *Token) {
	s.skip(isSpace)
	if s.isDone() || s.char == semicolon || s.char == newline || isComment(s.char) || isOperator(s.char) {
		return
	}
	t.Type = TokBlank
}

func (s *Scanner) scanVariable(t *Token) {
	s.readRune()
	var buf bytes.Buffer
	for !s.isDone() && isAlpha(s.char) {
		buf.WriteRune(s.char)
		s.readRune()
	}
	t.Literal = buf.String()
	t.Type = TokVariable
}

func (s *Scanner) scanComment(t *Token) {
	s.readRune()
	s.skip(isSpace)

	var buf bytes.Buffer
	for !s.isDone() && s.char != newline {
		buf.WriteRune(s.char)
		s.readRune()
	}
	t.Type = TokComment
	t.Literal = buf.String()
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

func (s *Scanner) skip(fn func(rune) bool) {
	for fn(s.char) {
		s.readRune()
	}
}

func (s *Scanner) isDone() bool {
	return s.curr >= len(s.buffer)
}

func (s *Scanner) switchSplit() {
	if s.isDone() || (s.char != squote && s.char != dquote) {
		return
	}
	defer s.readRune()
	if s.quote == dquote || s.quote == squote {
		s.quote = 0
		s.split = splitLiteral
		return
	}
	s.quote = s.char
	if s.quote == dquote {
		s.split = splitQuotedWeak
	} else {
		s.split = splitQuotedStrong
	}
}

func (s *Scanner) isQuoted() bool {
	return s.quote == dquote || s.quote == squote
}

func canEscape(q, r rune) bool {
	return (q == 0 || q == dquote) && r == backslash
}

func splitLiteral(r rune) bool {
	return isBlank(r) || isComment(r) || isQuote(r) || r == semicolon || isOperator(r)
}

func splitQuotedStrong(r rune) bool {
	return r == squote
}

func splitQuotedWeak(r rune) bool {
	return r == dquote || r == dollar
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

func isVar(r rune) bool {
	return r == dollar
}

func isQuote(r rune) bool {
	return r == dquote || r == squote
}

func isComment(r rune) bool {
	return r == pound
}

func isOperator(r rune) bool {
	return r == pipe || r == ampersand
}
