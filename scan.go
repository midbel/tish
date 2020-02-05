package tish

import (
	"bytes"
	"fmt"
	"io"
	// "strings"
	"unicode/utf8"
)

type ScanFunc func(*Scanner) ScanFunc

type Scanner struct {
	buffer []byte
	char   rune
	pos    int
	next   int

	queue chan Token
}

func NewScanner(str string) *Scanner {
	s := &Scanner{
		buffer: []byte(str),
		queue:  make(chan Token),
	}
	go s.run()
	return s
}

func (s *Scanner) Scan() (Token, error) {
	var err error
	tok, ok := <-s.queue
	if !ok {
		err = io.EOF
	}
	if tok.Type == tokError && err == nil {
		str := tok.Literal
		if s.pos < len(s.buffer) {
			str = fmt.Sprintf("%s: %s", str, s.buffer)
		}
		err = fmt.Errorf(str)
		s.drain()
	}
	return tok, err
}

func (s *Scanner) emit(str string, typof rune) {
	if len(str) == 0 {
		return
	}
	if typof == tokQuoted {
		str, typof = str[1:len(str)-1], tokWord
	}
	s.queue <- Token{Literal: str, Type: typof}
}

func (s *Scanner) emitTypeOf(typof rune) {
	s.queue <- Token{Type: typof}
}

func (s *Scanner) run() {
	defer close(s.queue)

	s.readRune()
	s.skip(isBlank)

	scan := scanDefault
	for scan != nil {
		scan = scan(s)
	}
}

func (s *Scanner) drain() {
	for range s.queue {
	}
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
	s.char, s.pos, s.next = r, s.next, s.next+n
}

func (s *Scanner) unreadRune() {
	if s.next <= 0 || s.char == 0 {
		return
	}

	s.next, s.pos = s.pos, s.pos-utf8.RuneLen(s.char)
	s.char, _ = utf8.DecodeRune(s.buffer[s.pos:])
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

// func (s *Scanner) skipBlanks() {
// 	for s.char == space || s.char == tab {
// 		s.readRune()
// 	}
// }

func scanDefault(s *Scanner) ScanFunc {
	var buf bytes.Buffer
	for !isDelim(s.char) {
		switch s.char {
		case pound:
			s.emit(buf.String(), tokWord)
			return scanComment
		case space, tab:
			s.emit(buf.String(), tokWord)
			return scanBlanks
		case dollar:
			s.emit(buf.String(), tokWord)
			return scanDollar
		case squote:
			s.emit(buf.String(), tokWord)
			buf.Reset()
			scanQuotedStrong(s)
			continue
		case dquote:
			s.emit(buf.String(), tokWord)
			buf.Reset()
			scanQuotedWeak(s)
		case lcurly:
			s.emit(buf.String(), tokWord)
			return scanBraces
		case backslash:
			s.readRune()
		default:
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	s.emit(buf.String(), tokWord)
	if s.char == tokEOF {
		s.emitTypeOf(s.char)
		return nil
	}
	return scanBlanks
}

func scanBraces(s *Scanner) ScanFunc {
	s.readRune()
	delim := func(r rune) bool {
		return r == rcurly || r == dot || r == comma || r == tokEOF
	}

	s.emitTypeOf(tokBeginBrace)
	for s.char != rcurly {
		switch {
		case s.char == tokEOF:
			s.emit("unterminated braces expression", tokError)
			return nil
		case s.char == space || s.char == tab:
			s.skip(isBlank)
		case s.char == dot:
			s.readRune()
			if s.char != dot {
				s.emit("unterminated sequence operator", tokError)
				return nil
			}
			s.emitTypeOf(tokSequence)
		case s.char == comma:
			s.emitTypeOf(comma)
		default:
			scanWord(s, delim)
			continue
		}
		s.readRune()
	}
	s.readRune()

	s.emitTypeOf(tokEndBrace)
	if s.char == tokEOF {
		s.emitTypeOf(s.char)
		return nil
	}
	return scanDefault
}

func scanComment(s *Scanner) ScanFunc {
	s.readRune()

	var buf bytes.Buffer
	for s.char != tokEOF {
		buf.WriteRune(s.char)
		s.readRune()
	}

	s.emit(buf.String(), tokComment)
	return nil
}

func scanDollar(s *Scanner) ScanFunc {
	s.readRune()
	switch s.char {
	case lparen:
		s.readRune()
		if s.char == lparen {
			s.readRune()
			return scanArithmetic
		}
		return scanSubstitution
	case lcurly:
		return scanParameter
	default:
		return scanVariable
	}
}

func scanParameter(s *Scanner) ScanFunc {
	s.readRune()
	var buf bytes.Buffer
	for s.char != rcurly {
		buf.WriteRune(s.char)
		s.readRune()
	}
	s.emit(buf.String(), tokVar)
	s.readRune()
	return scanDefault
}

func scanArithmetic(s *Scanner) ScanFunc {
	s.emitTypeOf(tokBeginArith)

	for {
		switch {
		case s.char == tokEOF:
			s.emit("unterminated arithmetic expression", tokError)
			return nil
		case isOperator(s.char):
			s.emitTypeOf(s.char)
		case isDigit(s.char):
			scanNumber(s)
			continue
		case s.char == rparen:
			if k := s.peekRune(); k == rparen {
				s.readRune()
				s.readRune()
				s.emitTypeOf(tokEndArith)
				return scanDefault
			}
			s.emit("unterminated arithmetic expression", tokError)
			return nil
		case s.char == lparen:
			scanGroup(s)
		case s.char == dollar:
			scanVariable(s)
			continue
		case s.char == space || s.char == tab:
			s.skip(isBlank)
			continue
		default:
		}
		s.readRune()
	}
}

func scanGroup(s *Scanner) {
	s.emitTypeOf(s.char)
	s.readRune()
	for s.char != rparen {
		switch {
		case isDigit(s.char):
			scanNumber(s)
			continue
		case isBlank(s.char):
			s.skip(isBlank)
			continue
		case isOperator(s.char):
			s.emitTypeOf(s.char)
		case s.char == dollar:
			scanVariable(s)
			continue
		case s.char == lparen:
			scanGroup(s)
		default:
		}
		s.readRune()
	}
	s.emitTypeOf(s.char)
}

func scanNumber(s *Scanner) {
	// if s.char == '0' {
	// 	switch peek := s.peekRune(); peek {
	// 	case 'x', 'X':
	// 	case 'o':
	// 	case 'b':
	// 	default:
	// 	}
	// }
	var buf bytes.Buffer
	for isDigit(s.char) {
		buf.WriteRune(s.char)
		s.readRune()
	}
	switch {
	case s.char == dot:
		buf.WriteRune(s.char)
		s.readRune()
		for isDigit(s.char) {
			buf.WriteRune(s.char)
			s.readRune()
		}
		s.emit(buf.String(), tokFloat)
	case isOperator(s.char) || isBlank(s.char) || s.char == rparen:
		s.emit(buf.String(), tokInteger)
	default:
		s.emit(fmt.Sprintf("number: invalid character: %c", s.char), tokError)
	}
}

func scanWord(s *Scanner, fn func(r rune) bool) {
	var buf bytes.Buffer
	for !fn(s.char) {
		if s.char == tokEOF {
			s.emit(fmt.Sprintf("unexpected end of string: %s", buf.String()), tokError)
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	s.emit(buf.String(), tokWord)
}

func scanSubstitution(s *Scanner) ScanFunc {
	s.emitTypeOf(tokBeginSub)
	for s.char != rparen {
		switch s.char {
		case tokEOF:
			s.emit("unterminated command subsitution", tokError)
			return nil
		case space, tab:
			scanBlanks(s)
		case squote:
			scanQuotedStrong(s)
		case dquote:
			scanQuotedWeak(s)
		case dollar:
			scanDollar(s)(s)
		case lcurly:
			scanBraces(s)
		default:
			scanDefault(s)
		}
	}
	s.readRune()
	s.emitTypeOf(tokEndSub)
	return scanDefault
}

func scanQuotedWeak(s *Scanner) ScanFunc {
	var buf bytes.Buffer
	// buf.WriteRune(dquote)

	s.readRune()
	for s.char != dquote {
		if s.char == tokEOF {
			s.emit(fmt.Sprintf("unterminated quoted string: %s", buf.String()), tokError)
			return nil
		}
		switch s.char {
		case dollar:
			s.emit(buf.String(), tokWord)
			buf.Reset()
			scanDollar(s)(s)
			continue
		case backslash:
			if k := s.peekRune(); k == dollar || k == dquote || k == backslash {
				s.readRune()
			}
		default:
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	s.readRune()

	// buf.WriteRune(dquote)
	// s.emit(buf.String(), tokQuoted)
	s.emit(buf.String(), tokWord)

	if s.char == tokEOF {
		s.emitTypeOf(s.char)
		return nil
	}
	return scanDefault
}

func scanQuotedStrong(s *Scanner) {
	var buf bytes.Buffer
	buf.WriteRune(squote)

	s.readRune()
	for s.char != squote {
		if s.char == tokEOF {
			s.emit(fmt.Sprintf("unterminated quoted string: %s", buf.String()), tokError)
			return
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	s.readRune()

	buf.WriteRune(squote)
	s.emit(buf.String(), tokQuoted)
}

func scanBlanks(s *Scanner) ScanFunc {
	s.skip(isBlank)
	s.emitTypeOf(tokBlank)
	return scanDefault
}

func scanVariable(s *Scanner) ScanFunc {
	if s.char == dollar {
		s.readRune()
	}

	var buf bytes.Buffer
	for isAlpha(s.char) {
		buf.WriteRune(s.char)
		s.readRune()
	}
	s.emit(buf.String(), tokVar)
	if s.char == tokEOF {
		s.emitTypeOf(tokEOF)
		return nil
	}
	return scanDefault
}

func isDelim(r rune) bool {
	return isBlank(r) || isMeta(r)
}

func isMeta(r rune) bool {
	return r == lparen || r == rparen || r == pipe ||
		r == semicolon || r == equal || r == ampersand
		// r == lcurly || r == rcurly
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

func isOperator(r rune) bool {
	return r == plus || r == div || r == minus || r == mul || r == modulo
}
