package parser

import (
	"bytes"
	"io"
	"unicode/utf8"

	"github.com/midbel/tish/internal/token"
)

var colonOps = map[rune]rune{
	minus:    token.ValIfUnset,
	plus:     token.ValIfSet,
	equal:    token.SetValIfUnset,
	question: token.ExitIfUnset,
	langle:   token.PadLeft,
	rangle:   token.PadRight,
}

var slashOps = map[rune]rune{
	slash:   token.ReplaceAll,
	percent: token.ReplaceSuffix,
	pound:   token.ReplacePrefix,
}

var testops = map[string]rune{
	// binary operators
	"-eq": token.Eq,
	"-ne": token.Ne,
	"-lt": token.Lt,
	"-le": token.Le,
	"-gt": token.Gt,
	"-ge": token.Ge,
	"-nt": token.NewerThan,
	"-ot": token.OlderThan,
	"-ef": token.SameFile,
	// unary operators
	"-e": token.FileExists,
	"-r": token.FileRead,
	"-h": token.FileLink,
	"-d": token.FileDir,
	"-w": token.FileWrite,
	"-s": token.FileSize,
	"-f": token.FileRegular,
	"-x": token.FileExec,
	"-z": token.StrNotEmpty,
	"-n": token.StrEmpty,
}

const (
	zero       = 0
	space      = ' '
	tab        = '\t'
	squote     = '\''
	dquote     = '"'
	dollar     = '$'
	pound      = '#'
	percent    = '%'
	slash      = '/'
	comma      = ','
	colon      = ':'
	minus      = '-'
	plus       = '+'
	question   = '?'
	underscore = '_'
	lcurly     = '{'
	rcurly     = '}'
	lparen     = '('
	rparen     = ')'
	lsquare    = '['
	rsquare    = ']'
	equal      = '='
	caret      = '^'
	ampersand  = '&'
	pipe       = '|'
	semicolon  = ';'
	langle     = '<'
	rangle     = '>'
	backslash  = '\\'
	dot        = '.'
	star       = '*'
	arobase    = '@'
	bang       = '!'
	nl         = '\n'
	cr         = '\r'
	tilde      = '~'
)

type Scanner struct {
	input []byte
	char  rune
	curr  int
	next  int

	str   bytes.Buffer
	state scanstack
}

func Scan(r io.Reader) *Scanner {
	buf, _ := io.ReadAll(r)
	s := Scanner{
		input: buf,
		state: defaultStack(),
	}
	s.read()
	return &s
}

func (s *Scanner) Scan() token.Token {
	s.reset()
	var tok token.Token
	if s.char == zero || s.char == utf8.RuneError {
		tok.Type = token.EOF
		return tok
	}
	if s.state.Arithmetic() {
		s.scanArithmetic(&tok)
		return tok
	}
	if s.state.Test() {
		s.scanTest(&tok)
		return tok
	}
	switch {
	case isBraces(s.char) && s.state.AcceptBraces():
		s.scanBraces(&tok)
	case isList(s.char) && s.state.Braces():
		s.scanList(&tok)
	case isOperator(s.char) && s.state.Expansion():
		s.scanOperator(&tok)
	case isBlank(s.char) && !s.state.Quoted():
		tok.Type = token.Blank
		s.skipBlank()
	case isSequence(s.char) && !s.state.Quoted():
		s.scanSequence(&tok)
	case isRedirectBis(s.char, s.peek()) && !s.state.Quoted():
		s.scanRedirect(&tok)
	case isAssign(s.char) && !s.state.Quoted():
		s.scanAssignment(&tok)
	case isDouble(s.char):
		s.scanQuote(&tok)
	case isSingle(s.char):
		s.scanString(&tok)
	case isComment(s.char):
		s.scanComment(&tok)
	case isVariable(s.char):
		s.scanDollar(&tok)
	case isTest(s.char, s.peek()):
		s.scanTest(&tok)
	default:
		s.scanLiteral(&tok)
	}
	return tok
}

func (s *Scanner) scanTest(tok *token.Token) {
	tok.Type = token.Invalid
	var skip bool
	switch k := s.peek(); {
	case s.char == lsquare && s.char == k:
		s.read()
		tok.Type = token.BegTest
		s.state.EnterTest()
	case s.char == rsquare && s.char == k:
		s.read()
		tok.Type = token.EndTest
		s.state.LeaveTest()
	case s.char == lparen:
		tok.Type = token.BegMath
	case s.char == rparen:
		tok.Type = token.EndMath
	case s.char == ampersand && k == s.char:
		tok.Type = token.And
		s.read()
	case s.char == pipe && k == s.char:
		tok.Type = token.Or
		s.read()
	case s.char == equal && k == s.char:
		tok.Type = token.Eq
		s.read()
	case s.char == bang && k == equal:
		tok.Type = token.Ne
		s.read()
	case s.char == langle:
		tok.Type = token.Lt
		if k == equal {
			s.read()
			tok.Type = token.Le
		}
	case s.char == rangle:
		tok.Type = token.Gt
		if k == equal {
			s.read()
			tok.Type = token.Ge
		}
	case s.char == bang:
		tok.Type = token.Not
	case isDouble(s.char):
		tok.Type = token.Quote
		s.state.ToggleQuote()
	case isSingle(s.char):
		s.scanString(tok)
		skip = true
	case isVariable(s.char):
		s.scanDollar(tok)
		skip = true
	case isBlank(s.char):
		tok.Type = token.Blank
	default:
		s.scanLiteral(tok)
		skip = true

		if k, ok := testops[tok.Literal]; ok {
			tok.Type = k
		}
	}
	if !skip {
		s.read()
	}
	if !s.state.Quoted() {
		s.skipBlank()
	}
}

func (s *Scanner) scanArithmetic(tok *token.Token) {
	s.skipBlank()
	switch {
	case isMath(s.char):
		s.scanMath(tok)
	case isDigit(s.char):
		s.scanDigit(tok)
	case isLetter(s.char):
		s.scanVariable(tok)
	default:
		tok.Type = token.Invalid
	}
}

func (s *Scanner) scanVariable(tok *token.Token) {
	tok.Type = token.Variable
	switch {
	case s.char == dollar:
		tok.Literal = "$"
		s.read()
	case s.char == pound:
		tok.Literal = "#"
		s.read()
	case s.char == question:
		tok.Literal = "?"
		s.read()
	case s.char == star:
		tok.Literal = "*"
		s.read()
	case s.char == arobase:
		tok.Literal = "@"
		s.read()
	case s.char == bang:
		tok.Literal = "!"
		s.read()
	case isDigit(s.char):
		for isDigit(s.char) {
			s.write()
			s.read()
		}
		tok.Literal = s.string()
	default:
		if !isLetter(s.char) {
			tok.Type = token.Invalid
			return
		}
		for isIdent(s.char) {
			s.write()
			s.read()
		}
		tok.Literal = s.string()
	}
}

func (s *Scanner) scanDigit(tok *token.Token) {
	for isDigit(s.char) {
		s.write()
		s.read()
	}
	if s.char == dot {
		s.write()
		s.read()
		for isDigit(s.char) {
			s.write()
			s.read()
		}
	}
	tok.Literal = s.string()
	tok.Type = token.Numeric
}

func (s *Scanner) scanMath(tok *token.Token) {
	switch s.char {
	case semicolon:
		tok.Type = token.List
	case caret:
		tok.Type = token.BitXor
	case tilde:
		tok.Type = token.BitNot
	case bang:
		tok.Type = token.Not
		if s.peek() == equal {
			tok.Type = token.Ne
			s.read()
		}
	case plus:
		tok.Type = token.Add
		if s.peek() == s.char {
			tok.Type = token.Inc
			s.read()
		}
	case minus:
		tok.Type = token.Sub
		if s.peek() == s.char {
			tok.Type = token.Dec
			s.read()
		}
	case star:
		tok.Type = token.Mul
		if s.peek() == s.char {
			tok.Type = token.Pow
			s.read()
		}
	case slash:
		tok.Type = token.Div
	case percent:
		tok.Type = token.Mod
	case lparen:
		tok.Type = token.BegMath
		s.state.EnterArithmetic()
	case rparen:
		tok.Type = token.EndMath
		s.state.LeaveArithmetic()
		if s.state.Depth() == 0 && s.peek() == s.char {
			s.read()
		}
	case pipe:
		tok.Type = token.BitOr
		if s.peek() == s.char {
			tok.Type = token.Or
			s.read()
		}
	case ampersand:
		tok.Type = token.BitAnd
		if s.peek() == s.char {
			tok.Type = token.And
			s.read()
		}
	case equal:
		tok.Type = token.Assign
		if s.peek() == s.char {
			s.read()
			tok.Type = token.Eq
		}
	case langle:
		tok.Type = token.Lt
		if s.peek() == equal {
			s.read()
			tok.Type = token.Le
			break
		}
		if s.peek() == s.char {
			s.read()
			tok.Type = token.LeftShift
		}
	case rangle:
		tok.Type = token.Gt
		if s.peek() == equal {
			s.read()
			tok.Type = token.Ge
			break
		}
		if s.peek() == s.char {
			s.read()
			tok.Type = token.RightShift
		}
	case question:
		tok.Type = token.Cond
	case colon:
		tok.Type = token.Alt
	default:
		tok.Type = token.Invalid
	}
	s.read()
}

func (s *Scanner) scanQuote(tok *token.Token) {
	tok.Type = token.Quote
	s.read()
	s.state.ToggleQuote()
	if s.state.Quoted() {
		return
	}
	s.skipBlankUntil(func(r rune) bool {
		return isSequence(r) || isAssign(r) || isComment(r) || isRedirectBis(r, s.peek())
	})
}

func (s *Scanner) scanBraces(tok *token.Token) {
	switch k := s.peek(); {
	case s.char == rcurly:
		tok.Type = token.EndBrace
		s.state.LeaveBrace()
	case s.char == lcurly && k != rcurly:
		tok.Type = token.BegBrace
		s.state.EnterBrace()
	default:
		s.scanLiteral(tok)
		return
	}
	s.read()
	s.skipBlank()
}

func (s *Scanner) scanList(tok *token.Token) {
	switch k := s.peek(); {
	case s.char == comma:
		tok.Type = token.Seq
	case s.char == dot && k == s.char:
		tok.Type = token.Range
		s.read()
	default:
	}
	if tok.Type == token.Invalid {
		return
	}
	s.read()
	s.skipBlank()
}

func (s *Scanner) scanAssignment(tok *token.Token) {
	tok.Type = token.Assign
	s.read()
	s.skipBlank()
}

func (s *Scanner) scanRedirect(tok *token.Token) {
	switch s.char {
	case langle:
		tok.Type = token.RedirectIn
	case rangle:
		tok.Type = token.RedirectOut
		if k := s.peek(); k == s.char {
			tok.Type = token.AppendOut
			s.read()
		}
	case ampersand:
		s.read()
		if s.char == rangle && s.peek() == s.char {
			s.read()
			tok.Type = token.AppendBoth
		} else if s.char == rangle {
			tok.Type = token.RedirectBoth
		} else {
			tok.Type = token.Invalid
		}
	case '0':
		s.read()
		if s.char != langle {
			tok.Type = token.Invalid
			break
		}
		tok.Type = token.RedirectIn
	case '1':
		s.read()
		if s.char == rangle && s.peek() == s.char {
			s.read()
			tok.Type = token.AppendOut
		} else if s.char == rangle {
			tok.Type = token.RedirectOut
		} else {
			tok.Type = token.Invalid
		}
	case '2':
		s.read()
		if s.char == rangle && s.peek() == s.char {
			s.read()
			tok.Type = token.AppendErr
		} else if s.char == rangle {
			tok.Type = token.RedirectErr
		} else {
			tok.Type = token.Invalid
		}
	default:
		tok.Type = token.Invalid
	}
	s.read()
	s.skipBlank()
}

func (s *Scanner) scanSequence(tok *token.Token) {
	switch k := s.peek(); {
	case s.char == semicolon:
		tok.Type = token.List
	case s.char == nl:
		tok.Type = token.List
		s.skipNL()
		return
	case s.char == ampersand && k == s.char:
		tok.Type = token.And
		s.read()
	case s.char == ampersand && isRedirect(k):
		s.scanRedirect(tok)
		return
	case s.char == pipe && k == s.char:
		tok.Type = token.Or
		s.read()
	case s.char == pipe && k == ampersand:
		tok.Type = token.PipeBoth
		s.read()
	case s.char == pipe:
		tok.Type = token.Pipe
	case s.char == rparen:
		tok.Type = token.EndSub
		if s.state.Substitution() {
			s.state.LeaveSubstitution()
		}
	case s.char == lparen:
		tok.Type = token.BegSub
	case s.char == comma:
		tok.Type = token.Comma
	default:
		tok.Type = token.Invalid
	}
	s.read()
	s.skipBlank()
}

func (s *Scanner) scanOperator(tok *token.Token) {
	if k := s.prev(); s.char == pound && k == lcurly {
		tok.Type = token.Length
		s.read()
		return
	}
	switch s.char {
	case rcurly:
		tok.Type = token.EndExp
		s.state.LeaveExpansion()
	case colon:
		tok.Type = token.Slice
		if t, ok := colonOps[s.peek()]; ok {
			s.read()
			tok.Type = t
		}
	case slash:
		tok.Type = token.Replace
		if t, ok := slashOps[s.peek()]; ok {
			s.read()
			tok.Type = t
		}
	case percent:
		tok.Type = token.TrimSuffix
		if k := s.peek(); k == percent {
			tok.Type = token.TrimSuffixLong
			s.read()
		}
	case pound:
		tok.Type = token.TrimPrefix
		if k := s.peek(); k == pound {
			tok.Type = token.TrimPrefixLong
			s.read()
		}
	case comma:
		tok.Type = token.Lower
		if k := s.peek(); k == comma {
			tok.Type = token.LowerAll
			s.read()
		}
	case caret:
		tok.Type = token.Upper
		if k := s.peek(); k == caret {
			tok.Type = token.UpperAll
			s.read()
		}
	default:
		tok.Type = token.Invalid
	}
	s.read()
}

func (s *Scanner) scanDollar(tok *token.Token) {
	s.read()
	if !s.state.Test() {
		if s.char == lcurly {
			tok.Type = token.BegExp
			s.state.EnterExpansion()
			s.read()
			return
		}
		if s.char == lparen && s.peek() == lparen {
			s.read()
			s.read()
			tok.Type = token.BegMath
			s.state.EnterArithmetic()
			return
		}
		if s.char == lparen {
			tok.Type = token.BegSub
			s.state.EnterSubstitution()
			s.read()
			return
		}
	}
	s.scanVariable(tok)
}

func (s *Scanner) scanComment(tok *token.Token) {
	s.read()
	s.skipBlank()
	for !s.done() && !isNL(s.char) {
		s.write()
		s.read()
	}
	if isNL(s.char) {
		s.read()
	}
	tok.Type = token.Comment
	tok.Literal = s.string()
}

func (s *Scanner) scanString(tok *token.Token) {
	s.read()
	for !isSingle(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Type = token.Literal
	tok.Literal = s.string()
	if !isSingle(s.char) {
		tok.Type = token.Invalid
	}
	s.read()
	if s.state.Test() {
		return
	}
	s.skipBlankUntil(func(r rune) bool {
		return isSequence(r) || isAssign(r) || isComment(r) || isRedirectBis(r, s.peek())
	})
}

func (s *Scanner) scanLiteral(tok *token.Token) {
	if s.state.Quoted() {
		s.scanQuotedLiteral(tok)
		return
	}
	for !s.done() && !s.stopLiteral(s.char) {
		if s.char == backslash && canEscape(s.peek()) {
			s.read()
		}
		s.write()
		s.read()
	}
	tok.Type = token.Literal
	tok.Literal = s.string()
	if token.IsKeyword(tok.Literal) {
		tok.Type = token.Keyword
		s.skipBlank()
	}
	if s.state.Test() {
		return
	}
	s.skipBlankUntil(func(r rune) bool {
		return isSequence(r) || isAssign(r) || isComment(r) || isRedirectBis(r, s.peek())
	})
}

func (s *Scanner) scanQuotedLiteral(tok *token.Token) {
	for !s.done() {
		if isDouble(s.char) || isVariable(s.char) {
			break
		}
		if s.state.Expansion() && isOperator(s.char) {
			break
		}
		s.write()
		s.read()
	}
	tok.Type = token.Literal
	tok.Literal = s.string()
}

func (s *Scanner) reset() {
	s.str.Reset()
}

func (s *Scanner) write() {
	s.str.WriteRune(s.char)
}

func (s *Scanner) string() string {
	return s.str.String()
}

func (s *Scanner) peek() rune {
	r, _ := utf8.DecodeRune(s.input[s.next:])
	return r
}

func (s *Scanner) prev() rune {
	r, _ := utf8.DecodeLastRune(s.input[:s.curr])
	return r
}

func (s *Scanner) read() {
	if s.curr >= len(s.input) {
		s.char = 0
		return
	}
	r, n := utf8.DecodeRune(s.input[s.next:])
	if r == utf8.RuneError {
		s.char = 0
		s.next = len(s.input)
	}
	s.char, s.curr, s.next = r, s.next, s.next+n
}

func (s *Scanner) done() bool {
	return s.char == zero || s.char == utf8.RuneError
}

func (s *Scanner) skipNL() {
	for isNL(s.char) {
		s.read()
	}
}

func (s *Scanner) skipBlank() {
	for isBlank(s.char) {
		s.read()
	}
}

func (s *Scanner) skipBlankUntil(fn func(rune) bool) {
	if !isBlank(s.char) {
		return
	}
	var (
		curr = s.curr
		next = s.next
		char = s.char
	)
	s.skipBlank()
	if !fn(s.char) {
		s.curr = curr
		s.next = next
		s.char = char
	}
}

func (s *Scanner) stopLiteral(r rune) bool {
	if s.state.Braces() && (s.char == dot || s.char == comma || s.char == rcurly) {
		return true
	}
	if s.state.Expansion() && isOperator(r) {
		return true
	}
	if s.char == lcurly {
		return s.peek() != rcurly
	}
	if isTest(s.char, s.peek()) {
		return true
	}
	ok := isBlank(s.char) || isSequence(s.char) || isDouble(s.char) ||
		isVariable(s.char) || isAssign(s.char)
	return ok
}

func canEscape(r rune) bool {
	return r == backslash || r == semicolon || r == dquote || r == dollar
}

func isBlank(r rune) bool {
	return r == space || r == tab
}

func isDouble(r rune) bool {
	return r == dquote
}

func isSingle(r rune) bool {
	return r == squote
}

func isQuote(r rune) bool {
	return isDouble(r) || isSingle(r)
}

func isVariable(r rune) bool {
	return r == dollar
}

func isComment(r rune) bool {
	return r == pound
}

func isIdent(r rune) bool {
	return isLetter(r) || isDigit(r) || r == underscore
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isOperator(r rune) bool {
	switch r {
	case caret, pound, colon, slash, percent, comma, rcurly:
		return true
	default:
		return false
	}
}

func isSequence(r rune) bool {
	switch r {
	case comma, ampersand, pipe, semicolon, rparen, lparen, nl:
		return true
	default:
		return false
	}
}

func isAssign(r rune) bool {
	return r == equal
}

func isTest(r, n rune) bool {
	return (r == lsquare && r == n) || (r == rsquare && r == n)
}

func isRedirect(r rune) bool {
	return r == langle || r == rangle
}

func isRedirectBis(r, k rune) bool {
	if isRedirect(r) {
		return true
	}
	switch {
	case r == ampersand && k == rangle:
	case r == '0' && k == langle:
	case r == '1' && k == rangle:
	case r == '2' && k == rangle:
	default:
		return false
	}
	return true
}

func isBraces(r rune) bool {
	return r == lcurly || r == rcurly
}

func isList(r rune) bool {
	return r == comma || r == dot
}

func isNL(r rune) bool {
	return r == cr || r == nl
}

func isMath(r rune) bool {
	switch r {
	case lparen, rparen, plus, minus, star, slash, percent, langle, rangle, equal, bang, ampersand, pipe, question, colon, caret, semicolon, tilde:
		return true
	default:
		return false
	}
}

type scanState int8

const (
	scanDefault scanState = iota
	scanQuote
	scanSub
	scanExp
	scanBrace
	scanMath
	scanTest
)

func (s scanState) String() string {
	switch s {
	default:
		return "unknown"
	case scanDefault:
		return "default"
	case scanQuote:
		return "quote"
	case scanSub:
		return "substitution"
	case scanExp:
		return "expansion"
	case scanBrace:
		return "braces"
	case scanMath:
		return "arithmetic"
	case scanTest:
		return "test"
	}
}

type scanstack []scanState

func defaultStack() scanstack {
	var s scanstack
	s.Push(scanDefault)
	return s
}

func (s *scanstack) Test() bool {
	return s.Curr() == scanTest
}

func (s *scanstack) EnterTest() {
	s.Push(scanTest)
}

func (s *scanstack) LeaveTest() {
	if s.Test() {
		s.Pop()
	}
}

func (s *scanstack) Quoted() bool {
	return s.Curr() == scanQuote
}

func (s *scanstack) ToggleQuote() {
	if s.Quoted() {
		s.Pop()
		return
	}
	s.Push(scanQuote)
}

func (s *scanstack) Expansion() bool {
	return s.Curr() == scanExp
}

func (s *scanstack) EnterExpansion() {
	s.Push(scanExp)
}

func (s *scanstack) LeaveExpansion() {
	if s.Expansion() {
		s.Pop()
	}
}

func (s *scanstack) Arithmetic() bool {
	return s.Curr() == scanMath
}

func (s *scanstack) Depth() int {
	var depth int
	for i := len(*s) - 1; i >= 1; i-- {
		if (*s)[i] != scanMath || ((*s)[i] == scanMath && (*s)[i-1] != scanMath) {
			break
		}
		depth++
	}
	return depth
}

func (s *scanstack) EnterArithmetic() {
	s.Push(scanMath)
}

func (s *scanstack) LeaveArithmetic() {
	if s.Arithmetic() {
		s.Pop()
	}
}

func (s *scanstack) Substitution() bool {
	return s.Curr() == scanSub
}

func (s *scanstack) EnterSubstitution() {
	s.Push(scanSub)
}

func (s *scanstack) LeaveSubstitution() {
	if s.Substitution() {
		s.Pop()
	}
}

func (s *scanstack) Braces() bool {
	return s.Curr() == scanBrace
}

func (s *scanstack) AcceptBraces() bool {
	return !s.Quoted() && !s.Expansion()
}

func (s *scanstack) EnterBrace() {
	s.Push(scanBrace)
}

func (s *scanstack) LeaveBrace() {
	if s.Braces() {
		s.Pop()
	}
}

func (s *scanstack) Default() bool {
	curr := s.Curr()
	return curr == scanDefault || curr == scanSub
}

func (s *scanstack) Pop() {
	n := s.Len()
	if n == 0 {
		return
	}
	n--
	if n >= 0 {
		*s = (*s)[:n]
	}
}

func (s *scanstack) Push(st scanState) {
	*s = append(*s, st)
}

func (s *scanstack) Len() int {
	return len(*s)
}

func (s *scanstack) Curr() scanState {
	n := s.Len()
	if n == 0 {
		return scanDefault
	}
	n--
	return (*s)[n]
}

func (s *scanstack) Prev() scanState {
	n := s.Len()
	n--
	n--
	if n >= 0 {
		return (*s)[n]
	}
	return scanDefault
}
