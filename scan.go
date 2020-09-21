package tish

import (
	"bytes"
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
	kwElif     = "elif"
	kwIn       = "in"
	kwReturn   = "return"
	kwFunc     = "function"
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
	kwElif,
	kwIn,
	kwReturn,
	kwFunc,
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
	caret      = '^'
	comma      = ','
)

type ScanFunc func(*Scanner) ScanFunc

const (
	stateCmdReset = iota
	stateFstBlank
	stateCmd
	stateSndBlank
	stateWait
)

type ScannerState struct {
	simple int
	quoted int
}

func (s *ScannerState) enterQuote() {
	s.quoted++
}

func (s *ScannerState) leaveQuote() {
	s.quoted--
}

func (s *ScannerState) isQuoted() bool {
	return s.quoted > 0
}

func (s *ScannerState) update(k Kind) {
	if k == TokSemicolon {
		s.reset()
		return
	}
	if !s.canAssign() {
		return
	}
	if k == TokBlank && (s.simple == stateCmdReset || s.simple == stateCmd) {
		s.simple++
	} else if (k == TokLiteral || k == TokVariable) && s.simple == stateFstBlank {
		s.simple++
	} else if (k == TokBegExp || k == TokBegArith) && s.simple == stateFstBlank {
		s.simple = stateWait
	} else if (k == TokEndExp || k == TokEndArith) && s.simple == stateWait {
		s.simple = stateCmd
	} else {
		s.simple = stateCmdReset
	}
}

func (s *ScannerState) canAssign() bool {
	return s.simple < stateSndBlank
}

func (s *ScannerState) reset() {
	s.simple = stateFstBlank
	s.quoted = 0
}

type Scanner struct {
	buffer []byte
	char   rune
	curr   int
	next   int

	ScannerState
	queue chan Token

	line   int
	column int
}

func NewScanner(r io.Reader) (*Scanner, error) {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var s Scanner
	s.buffer = bytes.ReplaceAll(buf, []byte{carriage, newline}, []byte{newline})
	s.queue = make(chan Token)
	s.line = 1
	s.reset()

	go s.scan()
	return &s, nil
}

func (s *Scanner) Scan() Token {
	tok, ok := <-s.queue
	if !ok {
		tok.Type = TokEOF
	}
	s.update(tok.Type)
	return tok
}

func (s *Scanner) scan() {
	defer close(s.queue)
	s.readRune()
	s.skip(isSpace)

	var scan ScanFunc
	for !s.isDone() {
		if scan == nil {
			switch {
			case isQuote(s.char):
				scan = scanQuote
			case isComment(s.char):
				scan = scanComment
			case isVar(s.char):
				scan = scanDollar
			case isBlank(s.char):
				scan = scanBlank
			case isControl(s.char):
				scan = scanControl
			default:
				scan = scanLiteral
			}
		}
		scan = scan(s)
	}
}

func (s *Scanner) isAssign() bool {
	// if s.char != equal || isSpace(s.prevRune()) || isSpace(s.nextRune()) {
	if s.char != equal || isSpace(s.prevRune()) {
		return false
	}
	return s.canAssign()
}

func (s *Scanner) emit(str string, kind Kind) {
	if str == "" {
		return
	}
	s.emitToken(str, kind)
}

func (s *Scanner) emitType(kind Kind) {
	s.emitToken("", kind)
}

func (s *Scanner) emitToken(str string, kind Kind) {
	tok := Token{
		Literal: str,
		Type:    kind,
		Position: Position{
			Line: s.line,
			Col:  s.column,
		},
	}
	if str != "" && tok.Type != TokInvalid {
		tok.Quoted = s.isQuoted()
	}
	s.queue <- tok
}

func (s *Scanner) readRune() {
	if s.char == newline {
		s.column = 0
	}
	c, z := utf8.DecodeRune(s.buffer[s.next:])
	if c == utf8.RuneError {
		if z == 0 {
			s.curr = s.next
		}
		return
	}
	s.char, s.curr, s.next = c, s.next, s.next+z

	s.column++
	if s.char == newline {
		s.line++
	}
}

func (s *Scanner) nextRune() rune {
	r, _ := utf8.DecodeRune(s.buffer[s.next:])
	return r
}

func (s *Scanner) prevRune() rune {
	r, _ := utf8.DecodeLastRune(s.buffer[:s.curr])
	return r
}

func (s *Scanner) skip(fn func(rune) bool) {
	for fn(s.char) {
		s.readRune()
	}
}

func (s *Scanner) isDone() bool {
	return s.curr >= len(s.buffer)
}

func scanUntil(s *Scanner, fn func(rune) bool) {
	var buf bytes.Buffer
	for !s.isDone() && !fn(s.char) {
		buf.WriteRune(s.char)
		s.readRune()
	}
	k := TokLiteral
	if s.isDone() && !fn(s.char) {
		k = TokInvalid
	}
	s.emit(buf.String(), k)
}

func scanLiteral(s *Scanner) ScanFunc {
	isDelim := func(r rune) bool {
		return isBlank(r) || isComment(r) || isQuote(r) || isControl(r) || isVar(r)
	}
	var buf bytes.Buffer
	for !s.isDone() && !isDelim(s.char) {
		if s.isAssign() {
			s.emit(buf.String(), TokLiteral)
			s.emitType(TokAssign)
			s.readRune()
			buf.Reset()
			continue
		}
		if isEscape(s.char) {
			s.readRune()
			if s.char == newline {
				s.readRune()
				s.skip(isSpace)
			}
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	var (
		k = TokLiteral
		x = sort.SearchStrings(keywords, buf.String())
	)
	if x < len(keywords) && keywords[x] == buf.String() {
		k = TokKeyword
		s.skip(isBlank)
	}
	s.emit(buf.String(), k)
	return nil
}

func scanQuote(s *Scanner) ScanFunc {
	s.enterQuote()
	defer s.leaveQuote()

	escape := func(r rune) bool {
		return r == dollar || r == backslash || r == dquote
	}

	var (
		quote = s.char
		kind  = TokLiteral
		buf   bytes.Buffer
	)

	isDelim := func(r rune) bool {
		return r == quote || r == newline
	}

	s.readRune()
	for !s.isDone() && !isDelim(s.char) {
		if quote == dquote {
			switch {
			case isVar(s.char):
				s.emit(buf.String(), TokLiteral)
				buf.Reset()
				scanDollar(s)(s)
			case isEscape(s.char) && escape(s.nextRune()):
				s.readRune()
			default:
			}
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	if s.isDone() || s.char != quote {
		kind = TokInvalid
	}
	s.readRune()
	s.emit(buf.String(), kind)
	return nil
}

func scanNumber(s *Scanner) ScanFunc {
	var buf bytes.Buffer
	for !s.isDone() && isDigit(s.char) {
		buf.WriteRune(s.char)
		s.readRune()
	}
	s.emit(buf.String(), TokNumber)
	return nil
}

func scanVariable(s *Scanner) ScanFunc {
	if isVar(s.char) {
		s.readRune()
	}
	var buf bytes.Buffer
	for !s.isDone() && isAlpha(s.char) {
		buf.WriteRune(s.char)
		s.readRune()
	}
	s.emit(buf.String(), TokVariable)
	return nil
}

func scanComment(s *Scanner) ScanFunc {
	if isComment(s.char) {
		s.readRune()
	}
	s.skip(isSpace)

	var buf bytes.Buffer
	for !s.isDone() && s.char != newline {
		buf.WriteRune(s.char)
		s.readRune()
	}
	s.emit(buf.String(), TokComment)
	return nil
}

func scanBraces(s *Scanner) ScanFunc {
	s.emitType(TokBegBrace)
	s.emitType(TokEndBrace)
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
		s.emitType(TokInvalid)
	case lcurly:
		s.readRune()
		return scanExpansion
	default:
		return scanVariable
	}
	return nil
}

func scanSlice(s *Scanner) bool {
	s.emitType(TokSlice)
	switch {
	case isVar(s.char):
		scanVariable(s)
	case isDigit(s.char):
		scanNumber(s)
	case s.char == colon:
		s.emit("0", TokNumber)
	default:
		s.emitType(TokInvalid)
		return false
	}
	if s.char != colon {
		return false
	}
	s.readRune()
	switch {
	case isVar(s.char):
		scanVariable(s)
	case isDigit(s.char):
		scanNumber(s)
	case s.char == rcurly:
		s.emit("0", TokNumber)
	default:
		return false
	}
	return true
}

func scanExpansion(s *Scanner) ScanFunc {
	s.emitType(TokBegExp)
	if s.char == pound {
		s.readRune()
		s.emitType(TokLen)
		scanVariable(s)
		if s.char != rcurly {
			s.emitType(TokInvalid)
			return nil
		}
		s.emitType(TokEndExp)
		s.readRune()
		return nil
	}
	scanVariable(s)
	switch s.char {
	case colon:
		s.readRune()
		if !scanSlice(s) {
			s.emitType(TokInvalid)
			return nil
		}
	case tilde:
		s.readRune()
		k := TokReverse
		if s.char == tilde {
			s.readRune()
			k = TokReverseAll
		}
		s.emitType(k)
	case comma:
		s.readRune()
		k := TokLower
		if s.char == comma {
			s.readRune()
			k = TokLowerAll
		}
		s.emitType(k)
	case caret:
		s.readRune()
		k := TokUpper
		if s.char == caret {
			s.readRune()
			k = TokUpperAll
		}
		s.emitType(k)
	case slash:
		s.readRune()
		switch s.char {
		case slash:
			s.readRune()
			s.emitType(TokReplaceAll)
		case percent:
			s.readRune()
			s.emitType(TokReplaceSuffix)
		case pound:
			s.readRune()
			s.emitType(TokReplacePrefix)
		default:
			if !isVar(s.char) && !isAlpha(s.char) {
				s.emitType(TokInvalid)
				return nil
			}
			s.emitType(TokReplace)
		}
		if isVar(s.char) {
			scanVariable(s)
		} else {
			scanUntil(s, func(r rune) bool { return r == slash })
		}
		if s.char != slash {
			s.emitType(TokInvalid)
			return nil
		}
		s.readRune()
		if isVar(s.char) {
			scanVariable(s)
		} else {
			scanUntil(s, func(r rune) bool { return r == rcurly })
		}
	case pound:
		s.readRune()
		k := TokTrimPrefix
		if s.char == pound {
			s.readRune()
			k = TokTrimPrefixLong
		}
		s.emitType(k)
		if s.char == dollar {
			scanVariable(s)
		} else {
			scanUntil(s, func(r rune) bool { return r == rcurly })
		}
	case percent:
		s.readRune()
		k := TokTrimSuffix
		if s.char == percent {
			s.readRune()
			k = TokTrimSuffixLong
		}
		s.emitType(k)
		if s.char == dollar {
			scanVariable(s)
		} else {
			scanUntil(s, func(r rune) bool { return r == rcurly })
		}
	case rcurly:
	default:
		s.emitType(TokInvalid)
		return nil
	}
	if s.char != rcurly {
		s.emitType(TokInvalid)
		return nil
	}
	s.readRune()
	s.emitType(TokEndExp)
	return nil
}

func scanArithmetic(s *Scanner) ScanFunc {
	s.emitType(TokBegArith)
	for !s.isDone() {
		if peek := s.nextRune(); s.char == rparen && peek == s.char {
			s.readRune()
			s.readRune()
			s.emitType(TokEndArith)
			return nil
		}
		scanExpression(s)
	}
	s.emitType(TokInvalid)
	return nil
}

func scanGroup(s *Scanner) ScanFunc {
	s.emitType(TokBegGroup)
	for !s.isDone() {
		if s.char == rparen {
			s.readRune()
			s.emitType(TokEndGroup)
			return nil
		}
		scanExpression(s)
	}
	s.emitType(TokInvalid)
	return nil
}

func scanExpression(s *Scanner) {
	switch {
	case isSpace(s.char):
		s.skip(isSpace)
	case isDigit(s.char):
		scanNumber(s)
	case isVar(s.char):
		scanVariable(s)
	case s.char == plus:
		s.readRune()
		s.emitType(TokAdd)
	case s.char == minus:
		s.readRune()
		s.emitType(TokSub)
	case s.char == star:
		s.readRune()
		s.emitType(TokMul)
	case s.char == slash:
		s.readRune()
		s.emitType(TokDiv)
	case s.char == percent:
		s.readRune()
		s.emitType(TokMod)
	case s.char == langle:
		s.readRune()
		if s.char == langle {
			s.readRune()
			s.emitType(TokLeftShift)
		}
	case s.char == rangle:
		s.readRune()
		if s.char == rangle {
			s.readRune()
			s.emitType(TokRightShift)
		}
	case s.char == lparen:
		s.readRune()
		scanGroup(s)
	default:
	}
}

func scanTest(s *Scanner) ScanFunc {
	return nil
}

func scanControl(s *Scanner) ScanFunc {
	switch s.char {
	case lcurly:
		s.readRune()
		scanBraces(s)
	case lsquare:

	case lparen:
		s.readRune()
		s.emitType(TokBegGroup)
	case rparen:
		s.readRune()
		s.emitType(TokEndGroup)
	case ampersand:
		s.readRune()
		if s.char == ampersand {
			s.readRune()
			s.emitType(TokAnd)
		} else {
			s.emitType(TokBackground)
		}
		s.skip(isSpace)
	case pipe:
		s.readRune()
		if s.char == pipe {
			s.readRune()
			s.emitType(TokOr)
		} else {
			s.emitType(TokPipe)
		}
		s.skip(isSpace)
	case semicolon:
		s.readRune()
		switch s.char {
		case semicolon:
			s.readRune()
			if s.char == ampersand {
				s.readRune()
				s.emitType(TokFallthrough)
			} else {
				s.emitType(TokBreak)
			}
		case ampersand:
			s.readRune()
			s.emitType(TokContinue)
		default:
			s.emitType(TokSemicolon)
		}
		s.skip(isBlank)
	default:
		s.emitType(TokInvalid)
	}
	return nil
}

func scanBlank(s *Scanner) ScanFunc {
	if s.char == newline {
		s.readRune()
		s.emitType(TokSemicolon)
		return nil
	}
	s.skip(isSpace)
	if s.isDone() || isComment(s.char) || isControl(s.char) {
		return nil
	}
	s.emitType(TokBlank)
	return nil
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
	return r == space || r == tab || r == utf8.RuneError
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

func isEscape(r rune) bool {
	return r == backslash
}

func isControl(r rune) bool {
	return isGroup(r) || r == semicolon || r == pipe || r == ampersand
}

func isGroup(r rune) bool {
	return r == lparen || r == rparen || r == lcurly || r == rcurly //|| r == lsquare || r == rsquare
}
