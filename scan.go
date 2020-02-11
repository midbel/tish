package tish

import (
	"bytes"
	"fmt"
	"io"
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
		err, tok = io.EOF, eof
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

func (s *Scanner) restore(pos int) {
	if pos < 0 || pos >= len(s.buffer) {
		return
	}
	r, n := utf8.DecodeRune(s.buffer[pos:])
	s.char, s.pos, s.next = r, pos, pos+n
}

func (s *Scanner) skip(fn func(rune) bool) {
	for fn(s.char) {
		s.readRune()
		if s.char == tokEOF {
			break
		}
	}
}

func scanDefault(s *Scanner) ScanFunc {
	delim := func(r rune) bool {
		return isComment(r) || isBlank(r) || isQuote(r) || isControl(r) ||
			r == dollar || r == lcurly || r == equal
	}
	for s.char != tokEOF {
		switch s.char {
		case pound:
			return scanComment
		case space, tab:
			return scanBlanks
		case dollar:
			return scanDollar
		case squote:
			scanQuotedStrong(s)
		case dquote:
			scanQuotedWeak(s)
		case lcurly:
			return scanBraces
		case lparen:
			return scanList
		case equal:
			s.readRune()
			s.emitTypeOf(equal)
		case semicolon:
			s.readRune()
			s.skip(isBlank)
			s.emitTypeOf(semicolon)
		case pipe:
			scanPipe(s)
			s.skip(isBlank)
		case ampersand:
			scanAmpersand(s)
			s.skip(isBlank)
		default:
			scanWord(s, delim)
		}
	}
	if s.char == tokEOF {
		s.emitTypeOf(s.char)
		return nil
	}
	return scanBlanks
}

func scanList(s *Scanner) ScanFunc {
	delim := func(r rune) bool {
		return isComment(r) || isBlank(r) || isQuote(r) || isControl(r) ||
			r == dollar || r == lcurly || r == equal
	}

	s.readRune()
	s.emitTypeOf(tokBeginList)
	for s.char != rparen {
		switch s.char {
		case tokEOF:
			s.emit("unterminated list", tokError)
			return nil
		case pound:
			s.emit("unterminated list", tokError)
			return nil
		case space, tab:
			scanBlanks(s)
		case dollar:
			scanDollar(s)
		case squote:
			scanQuotedStrong(s)
		case dquote:
			scanQuotedWeak(s)
		case lcurly:
			scanBraces(s)
		case lparen:
			scanList(s)
		case equal:
			s.readRune()
			s.emitTypeOf(equal)
		case semicolon:
			s.readRune()
			s.skip(isBlank)
			s.emitTypeOf(semicolon)
		case pipe:
			scanPipe(s)
			s.skip(isBlank)
		case ampersand:
			scanAmpersand(s)
			s.skip(isBlank)
		default:
			scanWord(s, delim)
		}
	}

	s.emitTypeOf(tokEndList)
	s.readRune()
	return scanDefault
}

// range: [prolog]{lower:upper}[epilog]
// list: [prolog]{item0,...,item1}[epilog]
func scanBraces(s *Scanner) ScanFunc {
	s.readRune()
	delim := func(r rune) bool {
		return r == lcurly || r == rcurly || r == dot || r == comma || r == tokEOF
	}

	s.emitTypeOf(tokBeginBrace)
	for s.char != rcurly {
		switch {
		case s.char == tokEOF:
			s.emit("unterminated braces expression", tokError)
			return nil
		case s.char == space || s.char == tab:
			s.skip(isBlank)
		case s.char == comma:
			s.emitTypeOf(comma)
		case s.char == lcurly:
			scanBraces(s)
			continue
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
	s.skip(isBlank)

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
		case s.char == langle:
			s.readRune()
			if s.char != langle {
				s.emit("invalid operator", tokError)
			}
			s.emitTypeOf(tokLeftShift)
		case s.char == rangle:
			s.readRune()
			if s.char != rangle {
				s.emit("invalid operator", tokError)
			}
			s.emitTypeOf(tokRightShift)
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
		s.emit(buf.String(), tokInt)
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
		if s.char == backslash {
			s.readRune()
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	s.emit(buf.String(), tokWord)
}

func scanSubstitution(s *Scanner) ScanFunc {
	delim := func(r rune) bool {
		return isComment(r) || isBlank(r) || isQuote(r) || isControl(r) ||
			r == dollar || r == lcurly || r == equal
	}
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
		case lparen:
			scanList(s)
		case equal:
			s.readRune()
			s.emitTypeOf(equal)
		case semicolon:
			s.readRune()
			s.skip(isBlank)
			s.emitTypeOf(semicolon)
		case pipe:
			scanPipe(s)
			s.skip(isBlank)
		case ampersand:
			scanAmpersand(s)
			s.skip(isBlank)
		default:
			scanWord(s, delim)
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
		case lcurly:
			s.emit(buf.String(), tokWord)
			buf.Reset()
			scanBraces(s)
			continue
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
	if s.char != ampersand && s.char != pipe {
		s.emitTypeOf(tokBlank)
	}
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

func scanPipe(s *Scanner) {
	s.readRune()
	switch s.char {
	case pipe:
		s.readRune()
		s.emitTypeOf(tokOr)
	default:
		s.emitTypeOf(pipe)
	}
}

func scanAmpersand(s *Scanner) {
	s.readRune()
	switch s.char {
	case ampersand:
		s.readRune()
		s.emitTypeOf(tokAnd)
	default:
		s.emitTypeOf(tokBackground)
	}
}

func isComment(r rune) bool {
	return r == pound
}

func isQuote(r rune) bool {
	return r == squote || r == dquote
}

func isControl(r rune) bool {
	return r == pipe || r == ampersand || r == semicolon || r == lparen || r == rparen
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
