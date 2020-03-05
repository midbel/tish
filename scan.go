package tish

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"unicode/utf8"
)

type ScanFunc func(*Scanner) ScanFunc

type Scanner struct {
	buffer []byte
	char   rune
	pos    int
	next   int

	queue chan Token

	quoted int

	line   int
	column int
	seen   int
}

func NewScanner(r io.Reader) *Scanner {
	str, _ := ioutil.ReadAll(r)
	s := &Scanner{
		buffer: bytes.ReplaceAll(str, []byte("\r\n"), []byte("\n")),
		queue:  make(chan Token),
		line:   1,
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
	tok.Position = Position{
		Line:   s.line,
		Column: s.column,
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

func (s *Scanner) enterQuote() {
	s.quoted++
}

func (s *Scanner) leaveQuote() {
	s.quoted--
}

func (s *Scanner) isQuoted() bool {
	return s.quoted > 0
}

func (s *Scanner) emit(str string, typof rune) {
	if len(str) == 0 {
		return
	}
	if typof == tokQuoted {
		str, typof = str[1:len(str)-1], tokWord
	}
	tok := Token{
		Literal: str,
		Type:    typof,
		Quoted:  s.isQuoted(),
	}
	if _, ok := keywords[tok.Literal]; ok && !tok.Quoted {
		tok.Type = tokKeyword
		s.skip(isBlank)
	}
	s.queue <- tok
}

func (s *Scanner) emitTypeOf(typof rune) {
	s.queue <- Token{
		Type:   typof,
		Quoted: s.isQuoted(),
	}
}

func (s *Scanner) run() {
	defer close(s.queue)

	s.readRune()
	skip := func(r rune) bool {
		return isBlank(r) || r == newline
	}
	s.skip(skip)

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

	if s.char == newline {
		s.line++
		s.seen, s.column = s.column, 0
	} else {
		s.column++
	}
}

func (s *Scanner) unreadRune() {
	if s.next <= 0 || s.char == 0 {
		return
	}

	if s.char == newline {
		s.line--
		s.column = s.seen
	} else {
		s.column--
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
			r == dollar || r == lcurly || r == equal || r == newline
	}
	for s.char != tokEOF {
		scanRedirections(s)
		switch s.char {
		case newline:
			s.emitTypeOf(semicolon)
			s.readRune()
			s.skip(func(r rune) bool {
				return isBlank(r) || r == newline
			})
		case backslash:
			if peek := s.peekRune(); peek == newline {
				s.readRune()
				s.readRune()
				s.skip(isBlank)
			}
		case pound:
			return scanComment
		case colon:
			s.readRune()
			if s.char == squote {
				return scanComment
			}
			s.emit("invalid syntax: missing single quote after colon", tokError)
			return nil
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
			return scanSubshell
		case equal:
			s.readRune()
			s.emitTypeOf(equal)
		case semicolon:
			s.readRune()
			s.skip(func(r rune) bool { return s.char == semicolon })
			if s.char == newline {
				s.skip(func(r rune) bool { return s.char == newline })
			}
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

func scanTest(s *Scanner) {

}

func scanRedirections(s *Scanner) {
	for {
		var (
			peek = s.peekRune()
			pos  = s.pos
			tok  rune
		)
		switch s.char {
		case langle:
			tok = tokRedirectStdin
		case rangle:
			switch peek {
			case ampersand:
				s.readRune()
				s.readRune()
				if s.char == '2' {
					tok = tokRedirectOutToErr
				} else {
					s.restore(pos)
					return
				}
			case rangle:
				s.readRune()
				tok = tokAppendStdout
			default:
				tok = tokRedirectStdout
			}
		case ampersand:
			if peek != rangle {
				return
			}
			s.readRune()
			if peek = s.peekRune(); peek == rangle {
				s.readRune()
				tok = tokAppendBoth
			} else {
				tok = tokRedirectBoth
			}
		case '2':
			if peek != rangle {
				return
			}
			s.readRune()
			switch peek = s.peekRune(); peek {
			case ampersand:
				s.readRune()
				s.readRune()
				if s.char == '1' {
					tok = tokRedirectErrToOut
				} else {
					s.restore(pos)
				}
			case rangle:
				s.readRune()
				tok = tokAppendStderr
			default:
				tok = tokRedirectStderr
			}
		default:
			return
		}
		s.readRune()
		s.emitTypeOf(tok)

		s.skip(isBlank)
	}
}

func scanSubshell(s *Scanner) ScanFunc {
	delim := func(r rune) bool {
		return isComment(r) || isBlank(r) || isQuote(r) || isControl(r) ||
			r == dollar || r == lcurly || r == equal
	}

	s.readRune()
	s.skip(func(r rune) bool { return isBlank(r) || r == newline })
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
			scanDollar(s)(s)
		case squote:
			scanQuotedStrong(s)
		case dquote:
			scanQuotedWeak(s)
		case lcurly:
			scanBraces(s)
		case lparen:
			scanSubshell(s)
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
		return r == lcurly || r == rcurly || r == comma || r == tokEOF
	}

	s.emitTypeOf(tokBeginBrace)
	for s.char != rcurly {
		switch {
		case s.char == tokEOF || s.char == semicolon || s.char == newline:
			s.emit("unterminated braces expression", tokError)
			return nil
		case s.char == space || s.char == tab:
			s.skip(isBlank)
			continue
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
	if s.char != rcurly {
		s.emit("unterminated braces expansion", tokError)
		return nil
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
	var delim func() bool
	if s.char == squote {
		delim = func() bool { return s.char == squote }
	} else {
		delim = func() bool { return s.char == tokEOF || s.char == newline }
	}
	s.readRune()
	s.skip(isBlank)

	var buf bytes.Buffer
	for {
		if delim() {
			break
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	if s.char == squote || s.char == newline {
		s.readRune()
	}
	s.skip(func(r rune) bool { return isBlank(s.char) || s.char == newline })
	s.emit(buf.String(), tokComment)
	s.emitTypeOf(semicolon)
	// return nil
	return scanDefault
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
	s.emitTypeOf(tokBeginParam)

	s.readRune()
	if s.char == pound {
		s.emitTypeOf(tokVarLength)
		s.readRune()

		scanVariable(s)
		if s.char != rcurly {
			s.emit("unterminated parameter expansion", tokError)
			return nil
		}
		s.readRune()
		s.emitTypeOf(tokEndParam)
		return scanDefault
	}
	scanVariable(s)
	switch s.char {
	case comma:
		s.readRune()
		if s.char == comma {
			s.readRune()
			s.emitTypeOf(tokLowerAll)
		} else {
			s.emitTypeOf(tokLower)
		}
	case caret:
		s.readRune()
		if s.char == caret {
			s.readRune()
			s.emitTypeOf(tokUpperAll)
		} else {
			s.emitTypeOf(tokUpper)
		}
	case modulo:
		s.readRune()
		if s.char == modulo {
			s.readRune()
			s.emitTypeOf(tokTrimSuffixLong)
		} else {
			s.emitTypeOf(tokTrimSuffix)
		}
		if s.char == dollar {
			scanDollar(s)(s)
		} else {
			scanWord(s, func(r rune) bool { return r == rcurly })
		}
	case pound:
		s.readRune()
		if s.char == pound {
			s.readRune()
			s.emitTypeOf(tokTrimPrefixLong)
		} else {
			s.emitTypeOf(tokTrimPrefix)
		}
		if s.char == dollar {
			scanDollar(s)(s)
		} else {
			scanWord(s, func(r rune) bool { return r == rcurly })
		}
	case slash:
		s.readRune()
		switch s.char {
		case slash:
			s.readRune()
			s.emitTypeOf(tokReplaceAll)
		case modulo:
			s.readRune()
			s.emitTypeOf(tokReplaceSuffix)
		case pound:
			s.readRune()
			s.emitTypeOf(tokReplacePrefix)
		default:
			s.emitTypeOf(tokReplace)
		}
		if s.char == dollar {
			scanDollar(s)(s)
		} else {
			scanWord(s, func(r rune) bool { return r == slash })
		}
		if s.char != slash {
			s.emit(fmt.Sprintf("invalid char in parameter expansion: '%c'", s.char), tokError)
			return nil
		}
		s.readRune()

		s.emitTypeOf(tokReplace)
		if s.char == dollar {
			scanDollar(s)(s)
		} else {
			scanWord(s, func(r rune) bool { return r == rcurly })
		}
	case colon:
		s.readRune()
		if s.char == minus || s.char == equal || s.char == plus {
			var typof rune
			switch s.char {
			case minus:
				typof = tokGetIfUndef
			case equal:
				typof = tokSetIfUndef
			case plus:
				typof = tokGetIfDef
			}
			s.readRune()
			s.emitTypeOf(typof)
			if s.char == dollar {
				scanDollar(s)(s)
			} else {
				scanWord(s, func(r rune) bool { return r == rcurly })
			}
		} else {
			scanSlice(s)
		}
	case minus:
		s.readRune()
		s.emitTypeOf(tokGetIfUndef)
		if s.char == dollar {
			scanDollar(s)(s)
		} else {
			scanWord(s, func(r rune) bool { return r == rcurly })
		}
	case equal:
		s.readRune()
		s.emitTypeOf(tokSetIfUndef)
		if s.char == dollar {
			scanDollar(s)(s)
		} else {
			scanWord(s, func(r rune) bool { return r == rcurly })
		}
	case plus:
		s.readRune()
		s.emitTypeOf(tokGetIfDef)
		if s.char == dollar {
			scanDollar(s)(s)
		} else {
			scanWord(s, func(r rune) bool { return r == rcurly })
		}
	case rcurly:
	default:
		s.emit(fmt.Sprintf("invalid char in parameter expansion: '%c'", s.char), tokError)
		return nil
	}
	if s.char != rcurly {
		s.emit("unterminated parameter expansion", tokError)
		return nil
	}
	s.readRune()

	s.emitTypeOf(tokEndParam)
	return scanDefault
}

func scanSlice(s *Scanner) {
	s.emitTypeOf(tokSliceOffset)
	s.skip(isBlank)

	scan := func() bool {
		closed := s.char == lparen
		if closed {
			s.readRune()
			s.skip(isBlank)
		}
		if s.char == dollar {
			scanDollar(s)(s)
		} else {
			scanNumber(s)
		}
		if closed {
			s.skip(isBlank)
			if s.char == rparen {
				s.readRune()
			} else {
				s.emit("unterminated parenthese in slice", tokError)
				return false
			}
		}
		return true
	}

	switch {
	case isDigit(s.char) || s.char == minus || s.char == lparen || s.char == dollar:
		if !scan() {
			return
		}
	case s.char == colon:
		s.emit("0", tokInt)
	default:
		s.emit(fmt.Sprintf("invalid char in slice: '%c'", s.char), tokError)
		return
	}
	if s.char == colon {
		s.readRune()
	}
	s.emitTypeOf(tokSliceLen)
	switch {
	case s.char == rcurly:
		s.emit("0", tokInt)
	case isDigit(s.char) || s.char == minus || s.char == lparen || s.char == dollar:
		if !scan() {
			return
		}
	default:
		s.emit(fmt.Sprintf("invalid char in slice: '%c'", s.char), tokError)
	}
}

func scanArithmetic(s *Scanner) ScanFunc {
	s.emitTypeOf(tokBeginArith)
	s.skip(func(r rune) bool { return isBlank(r) || r == newline })
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
			s.readRune()
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
			s.readRune()
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
	if s.char != minus && !isDigit(s.char) {
		s.emit(fmt.Sprintf("number: invalid char: '%c'", s.char), tokError)
		return
	}
	if s.char == minus {
		buf.WriteRune(s.char)
		s.readRune()
	}
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
	case isOperator(s.char) || isBlank(s.char) || s.char == rparen || s.char == rcurly || s.char == colon:
		s.emit(buf.String(), tokInt)
	default:
		s.emit(fmt.Sprintf("number: invalid char: '%c'", s.char), tokError)
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
	s.skip(func(r rune) bool { return isBlank(r) || r == newline })
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
			scanSubshell(s)
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
	s.enterQuote()
	defer s.leaveQuote()

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
	s.enterQuote()
	defer s.leaveQuote()

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
	switch s.char {
	case ampersand:
	case pipe:
	case langle:
	case rangle:
	case pound:
	default:
		if k := s.peekRune(); s.char == '2' && k == rangle {
			break
		}
		s.emitTypeOf(tokBlank)
	}
	return scanDefault
}

func scanVariable(s *Scanner) ScanFunc {
	if isInternal(s.char) {
		s.emit(string(s.char), tokVar)
		s.readRune()
		return scanDefault
	}
	if isDigit(s.char) {
		s.emit(fmt.Sprintf("invalid char in variable name: %c", s.char), tokError)
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
	case ampersand:
		s.readRune()
		s.emitTypeOf(tokPipeBoth)
	default:
		s.emitTypeOf(tokPipe)
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
	return r == plus || r == div || r == minus || r == mul ||
		r == modulo || r == pipe || r == ampersand ||
		r == langle || r == rangle
}

func isInternal(r rune) bool {
	return r == question || r == dollar || r == bang || r == pound || r == arobase
}
