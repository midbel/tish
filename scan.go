package tish

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	EOF rune = -(iota + 1)
	EOL
	Blank
	Literal
	Number
	Comment
	Variable
	Quote
	Keyword
	Expression
	Substitution
	Option
	Assign
	Not
	Add
	Sub
	Mul
	Div
	Mod
	Lshift
	Rshift
	Band
	Bor
	Eq
	Ne
	Lt
	Le
	Gt
	Ge
	And
	Or
	Pipe
	PipeBoth
	RedirectIn
	RedirectOut
	RedirectErr
	RedirectBoth
	RedirectErrOut
	RedirectOutErr
	AppendOut
	AppendErr
	AppendBoth
	BegTest
	EndTest
	BegList
	EndList
	BegExp
	EndExp
	BegExpr
	EndExpr
	RemSuffix
	RemLongSuffix
	RemPrefix
	RemLongPrefix
	Substring
	ReplaceMatch
	ReplaceAll
	ReplaceSuffix
	ReplacePrefix
	Lowercase
	LowercaseAll
	Uppercase
	UppercaseAll
	Invalid
)

type Token struct {
	Literal string
	Type    rune
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	case EOF:
		return "<EOF>"
	case EOL:
		return "<EOL>"
	case Blank:
		return "<BLANK>"
	case Assign:
		return "<ASSIGNMENT>"
	case Quote:
		return "<QUOTE>"
	case And:
		return "<AND>"
	case Or:
		return "<OR>"
	case Not:
		return "<NOT>"
	case Add:
		return "<ADD>"
	case Sub:
		return "<SUB>"
	case Mul:
		return "<MUL>"
	case Div:
		return "<DIV>"
	case Mod:
		return "<MOD>"
	case Band:
		return "<BAND>"
	case Bor:
		return "<BOR>"
	case Lshift:
		return "<LSHIFT>"
	case Rshift:
		return "<RSHIFT>"
	case Eq:
		return "<EQUAL>"
	case Ne:
		return "<NOT EQUAL>"
	case Lt:
		return "<LESSER THAN>"
	case Le:
		return "<LESSER EQUAL>"
	case Gt:
		return "<LESSER THAN>"
	case Ge:
		return "<GREATER EQUAL>"
	case Pipe:
		return "<PIPE>"
	case PipeBoth:
		return "<PIPE BOTH>"
	case RedirectIn:
		return "<REDIRECT IN>"
	case RedirectOut:
		return "<REDIRECT OUT>"
	case RedirectErr:
		return "<REDIRECT ERR>"
	case RedirectBoth:
		return "<REDIRECT BOTH>"
	case RedirectErrOut:
		return "<REDIRECT ERR OUT>"
	case RedirectOutErr:
		return "<REDIRECT OUT ERR>"
	case AppendOut:
		return "<APPEND OUT>"
	case AppendErr:
		return "<APPEND ERR>"
	case AppendBoth:
		return "<APPEND BOTH>"
	case BegTest:
		return "<BEGIN TEST>"
	case EndTest:
		return "<END TEST>"
	case BegList:
		return "<BEGIN LIST>"
	case EndList:
		return "<END LIST>"
	case BegExpr:
		return "<BEGIN EXPR>"
	case EndExpr:
		return "<END EXPR>"
	case BegExp:
		return "<BEGIN EXPANSION>"
	case EndExp:
		return "<END EXPANSION>"
	case RemSuffix:
		return "<REMOVE SUFFIX>"
	case RemLongSuffix:
		return "<REMOVE LONG SUFFIX>"
	case RemPrefix:
		return "<REMOVE PREFIX>"
	case RemLongPrefix:
		return "<REMOVE LONG PREFIX>"
	case Substring:
		return "<SUBSTRING>"
	case ReplaceMatch:
		return "<REPLACE>"
	case ReplaceAll:
		return "<REPLACE ALL>"
	case ReplaceSuffix:
		return "<REPLACE SUFFIX>"
	case ReplacePrefix:
		return "<REPLACE PREFIX>"
	case Lowercase:
		return "<LOWERCASE>"
	case LowercaseAll:
		return "<LOWERCASE ALL>"
	case Uppercase:
		return "<UPPERCASE>"
	case UppercaseAll:
		return "<UPPERCASE ALL>"
	case Keyword:
		prefix = "keyword"
	case Literal:
		prefix = "literal"
	case Number:
		prefix = "number"
	case Comment:
		prefix = "comment"
	case Variable:
		prefix = "variable"
	case Expression:
		prefix = "expression"
	case Substitution:
		prefix = "substitution"
	case Option:
		prefix = "option"
	case Invalid:
		prefix = "invalid"
	default:
		prefix = "unknown"
	}
	return fmt.Sprintf("%s(%s)", prefix, t.Literal)
}

type scanFunc func() Token

type scanState struct {
	keepBlank bool
	nested    bool
	scan      scanFunc
}

func createState(scan scanFunc, blank bool) scanState {
	return scanState{
		keepBlank: blank,
		scan:      scan,
	}
}

func nestedState(scan scanFunc, blank bool) scanState {
	return scanState{
		keepBlank: blank,
		nested:    true,
		scan:      scan,
	}
}

type cursor struct {
	char rune
	curr int
	next int
}

type Scanner struct {
	input []byte
	cursor
	old cursor

	quoted bool
	scan   []scanState

	str bytes.Buffer
}

func Scan(r io.Reader) (*Scanner, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	s := Scanner{
		input: buf,
	}
	s.pushFunc(s.scanDefault, true)
	s.read()
	s.skip(isBlank)
	return &s, nil
}

func (s *Scanner) Scan() Token {
	defer s.reset()

	var tok Token
	if s.done() {
		tok.Type = EOF
		return tok
	}
	if isBlank(s.char) && !s.keepBlank() {
		s.skip(isBlank)
	}
	scan := s.getFunc()
	return scan()
}

func (s *Scanner) pushState(state scanState) {
	s.scan = append(s.scan, state)
}

func (s *Scanner) pushFunc(scan scanFunc, blank bool) {
	s.pushState(createState(scan, blank))
}

func (s *Scanner) popFunc() bool {
	n := len(s.scan)
	if n == 0 {
		return false
	}
	state := s.scan[n-1]
	s.scan = s.scan[:n-1]
	return state.nested
}

func (s *Scanner) getFunc() scanFunc {
	n := len(s.scan)
	if n == 0 {
		return s.scanDefault
	}
	return s.scan[n-1].scan
}

func (s *Scanner) keepBlank() bool {
	n := len(s.scan)
	if n == 0 {
		return true
	}
	return s.scan[n-1].keepBlank
}

func (s *Scanner) scanDefault() Token {
	var tok Token
	switch k := s.peek(); {
	case isDollar(s.char):
		s.scanDollar(&tok)
	case isEOL(s.char):
		s.scanEOL(&tok)
	case isSpace(s.char):
		s.scanBlank(&tok)
	case isGroup(s.char):
		s.scanGroup(&tok)
	case isComment(s.char):
		s.scanComment(&tok)
	case isDouble(s.char):
		s.scanEnterQuote(&tok)
	case isSingle(s.char):
		s.scanString(&tok)
	case isRedirect(s.char, k):
		s.scanRedirection(&tok)
	case isLogical(s.char, k):
		s.scanLogical(&tok)
	case isAssign(s.char):
		s.scanAssign(&tok)
	case isTest(s.char, k):
		s.scanCdt(&tok)
	default:
		s.scanLiteral(&tok, isLiteral)
	}
	return tok
}

func (s *Scanner) scanTest() Token {
	var tok Token
	switch k := s.peek(); {
	case s.char == '-':
		s.scanOption(&tok)
	case isCmp(s.char):
		s.scanCompare(&tok)
	case isGroup(s.char):
		s.scanGroup(&tok)
	case isDollar(s.char):
		s.scanDollar(&tok)
	case isDouble(s.char):
		s.scanEnterQuote(&tok)
	case isSingle(s.char):
		s.scanString(&tok)
	case isSpace(s.char):
		s.scanBlank(&tok)
	case isLogical(s.char, k):
		s.scanLogical(&tok)
	case isTest(s.char, k):
		s.scanCdt(&tok)
	default:
		s.scanLiteral(&tok, isLiteral)
	}
	return tok
}

func (s *Scanner) scanGroup(tok *Token) {
	tok.Type = BegList
	if s.char == rparen {
		tok.Type = EndList
	}
	s.read()
}

func (s *Scanner) scanOption(tok *Token) {
	s.read()
	s.scanLiteral(tok, isLetter)
	if tok.Type == Literal {
		tok.Type = Option
	} else {
		tok.Type = Invalid
	}
}

func (s *Scanner) scanCdt(tok *Token) {
	if s.char == lsquare {
		tok.Type = BegTest
		s.pushFunc(s.scanTest, false)
	} else {
		tok.Type = EndTest
		s.popFunc()
	}
	s.read()
	s.read()
}

func (s *Scanner) scanCompare(tok *Token) {
	switch s.char {
	case bang:
		tok.Type = Not
		if k := s.peek(); k == equal {
			tok.Type = Ne
		}
	case equal:
		tok.Type = Eq
		if k := s.peek(); k == equal {
			s.read()
		}
	case langle:
		tok.Type = Lt
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = Le
		}
	case rangle:
		tok.Type = Gt
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = Ge
		}
	default:
		tok.Type = Invalid
	}
	s.read()
}

func (s *Scanner) scanAssign(tok *Token) {
	tok.Type = Assign
	s.read()
	s.skip(isSpace)
}

func (s *Scanner) scanLogical(tok *Token) {
	switch s.char {
	default:
		tok.Type = Invalid
	case pipe:
		tok.Type = Or
	case ampersand:
		tok.Type = And
	}
	s.read()
	s.read()
	s.skip(isSpace)
}

func (s *Scanner) scanRedirection(tok *Token) {
	switch s.char {
	case pipe:
		tok.Type = Pipe
		if s.peek() == ampersand {
			tok.Type = PipeBoth
			s.read()
		}
	case langle:
		tok.Type = RedirectIn
	case ampersand:
		tok.Type = RedirectBoth
		s.read()
		s.read()
		if s.char == rangle {
			tok.Type = AppendBoth
		}
	case '1', rangle:
		if s.char == '1' {
			s.read()
		}
		tok.Type = RedirectOut
		s.read()
		if k := s.peek(); s.char == rangle {
			tok.Type = AppendOut
			s.read()
		} else if s.char == ampersand && k == '2' {
			s.read()
			s.read()
			tok.Type = RedirectOutErr
		}
	case '2':
		s.read()
		s.read()
		tok.Type = RedirectErr
		if k := s.peek(); s.char == rangle {
			s.read()
			tok.Type = AppendErr
		} else if s.char == ampersand && k == '1' {
			s.read()
			s.read()
			tok.Type = RedirectErrOut
		}
	default:
		tok.Type = Invalid
	}
	s.read()
	s.skip(isBlank)
}

func (s *Scanner) scanString(tok *Token) {
	s.read()
	for !isSingle(s.char) && !s.done() {
		s.write(s.char)
		s.read()
	}
	tok.Type = Literal
	tok.Literal = s.literal()
	if !isSingle(s.char) {
		tok.Type = Invalid
	} else {
		s.read()
	}
}

func (s *Scanner) scanLiteral(tok *Token, accept func(rune) bool) {
	for accept(s.char) && !s.done() {
		s.write(s.char)
		s.read()
	}

	tok.Type = Literal
	tok.Literal = s.literal()
	if isKeyword(tok.Literal) {
		tok.Type = Keyword
	} else {
		s.skipBlankIfPossible()
	}
}

func (s *Scanner) scanNumber(tok *Token) {
	tok.Type = Number
	if s.char == minus {
		s.write(s.char)
		s.read()
	}
	for !s.done() && isDigit(s.char) {
		s.write(s.char)
		s.read()
	}
	tok.Literal = s.literal()
	if s.char != dot {
		return
	}
	s.write(s.char)
	s.read()
	for !s.done() && isDigit(s.char) {
		s.write(s.char)
		s.read()
	}
	tok.Literal = s.literal()
}

func (s *Scanner) scanDollar(tok *Token) {
	s.read()
	switch {
	case s.char == lparen && s.peek() == s.char:
		tok.Type = BegExpr
		s.read()
		s.read()
		s.pushFunc(s.scanExpr, false)
	case s.char == lparen:
		s.scanSubstitution(tok)
	case s.char == lcurly:
		tok.Type = BegExp
		s.pushFunc(s.scanExpansion, false)
		s.read()
	default:
		s.scanIdentifier(tok)
	}
}

func (s *Scanner) scanExpr() Token {
	var tok Token
	switch {
	case s.char == plus:
		tok.Type = Add
		s.read()
	case s.char == minus:
		tok.Type = Sub
		s.read()
	case s.char == star:
		tok.Type = Mul
		s.read()
	case s.char == slash:
		tok.Type = Div
		s.read()
	case s.char == percent:
		tok.Type = Mod
		s.read()
	case s.char == langle:
		tok.Type = Lt
		s.read()
		if s.char == equal {
			tok.Type = Le
			s.read()
		} else if s.char == langle {
			tok.Type = Lshift
			s.read()
		}
	case s.char == rangle:
		tok.Type = Gt
		s.read()
		if s.char == equal {
			tok.Type = Ge
			s.read()
		} else if s.char == rangle {
			tok.Type = Rshift
			s.read()
		}
	case s.char == lparen:
		tok.Type = BegList
		s.read()
		s.pushState(nestedState(s.scanExpr, false))
	case s.char == rparen:
		tok.Type = EndList
		s.read()
		if nested := s.popFunc(); !nested {
			tok.Type = EndExpr
			if s.char != rparen {
				tok.Type = Invalid
			}
			s.read()
		}
	case s.char == semicolon:
		tok.Type = EOL
		s.read()
	case s.char == equal:
		tok.Type = Assign
		s.read()
		if s.char == equal {
			tok.Type = Eq
			s.read()
		}
	case s.char == bang:
		tok.Type = Not
		s.read()
		if s.char == equal {
			tok.Type = Ne
			s.read()
		}
	case s.char == ampersand:
		tok.Type = Band
		s.read()
		if s.char == ampersand {
			tok.Type = And
			s.read()
		}
	case s.char == pipe:
		tok.Type = Bor
		s.read()
		if s.char == pipe {
			tok.Type = Or
			s.read()
		}
	case isDigit(s.char):
		s.scanNumber(&tok)
	default:
		s.scanIdentifier(&tok)
	}
	return tok
}

func (s *Scanner) scanExpansion() Token {
	var tok Token
	switch {
	case s.char == rcurly:
		s.read()
		tok.Type = EndExp
		s.popFunc()
	case s.char == lparen:
		s.read()
		tok.Type = BegList
	case s.char == rparen:
		s.read()
		tok.Type = EndList
	case s.char == comma:
		s.read()
		tok.Type = Lowercase
		if s.char == comma {
			s.read()
			tok.Type = LowercaseAll
		}
	case s.char == caret:
		s.read()
		tok.Type = Uppercase
		if s.char == caret {
			s.read()
			tok.Type = UppercaseAll
		}
	case s.char == percent:
		s.read()
		tok.Type = RemSuffix
		if s.char == percent {
			s.read()
			tok.Type = RemLongSuffix
		}
	case s.char == pound:
		s.read()
		tok.Type = RemPrefix
		if s.char == pound {
			s.read()
			tok.Type = RemLongPrefix
		}
	case s.char == colon:
		s.read()
		tok.Type = Substring
	case s.char == slash:
		s.read()
		tok.Type = ReplaceMatch
		if s.char == slash {
			s.read()
			tok.Type = ReplaceAll
		} else if s.char == percent {
			s.read()
			tok.Type = ReplaceSuffix
		} else if s.char == pound {
			s.read()
			tok.Type = ReplacePrefix
		}
	case isDouble(s.char):
		s.scanEnterQuote(&tok)
	case isSingle(s.char):
		s.scanString(&tok)
	case isDigit(s.char) || s.char == minus:
		s.scanNumber(&tok)
	default:
		s.scanLiteral(&tok, isAlpha)
	}
	return tok
}

func (s *Scanner) scanSubstitution(tok *Token) {
	var scan func(bool)
	scan = func(top bool) {
		s.read()
		for s.char != rparen && !s.done() {
			ch := s.char
			s.write(s.char)
			s.read()
			if isDollar(ch) && s.char == lparen {
				s.write(s.char)
				scan(false)
			}
		}
		if s.char == rparen && !top {
			s.write(s.char)
			s.read()
		}
	}

	scan(true)
	tok.Literal = s.literal()
	tok.Type = Substitution
	if s.char != rparen {
		tok.Type = Invalid
	} else {
		s.read()
	}
}

func (s *Scanner) scanIdentifier(tok *Token) {
	tok.Type = Variable
	if !isLetter(s.char) {
		tok.Type = Invalid
	}
	for isLetter(s.char) && !s.done() {
		s.write(s.char)
		s.read()
	}
	tok.Literal = s.literal()
}

func (s *Scanner) scanComment(tok *Token) {
	s.read()
	s.skip(isBlank)
	for !isNL(s.char) && !s.done() {
		s.write(s.char)
		s.read()
	}
	s.skip(isBlank)
	tok.Type = Comment
	tok.Literal = s.literal()
}

func (s *Scanner) scanQuoted() Token {
	var tok Token
	switch {
	case isDollar(s.char):
		s.scanDollar(&tok)
	case isDouble(s.char):
		s.scanLeaveQuote(&tok)
	default:
		s.scanLiteral(&tok, func(r rune) bool {
			return !isDollar(r) && !isDouble(r)
		})
	}
	return tok
}

func (s *Scanner) scanEnterQuote(tok *Token) {
	tok.Type = Quote
	s.read()
	s.pushFunc(s.scanQuoted, true)
}

func (s *Scanner) scanLeaveQuote(tok *Token) {
	tok.Type = Quote
	s.read()
	s.popFunc()
	s.skipBlankIfPossible()
}

func (s *Scanner) scanBlank(tok *Token) {
	tok.Type = Blank
	s.skip(isSpace)
}

func (s *Scanner) scanEOL(tok *Token) {
	tok.Type = EOL
	s.read()
	s.skip(isBlank)
}

func (s *Scanner) skipBlankIfPossible() {
	if !s.keepBlank() || !isBlank(s.char) {
		return
	}
	s.save()
	s.skip(isSpace)
	switch s.char {
	case equal:
	case pipe:
	case ampersand:
	case langle:
	case rangle:
	case lparen:
	case rparen:
	case semicolon:
	default:
		s.restore()
	}
}

func (s *Scanner) save() {
	s.old = s.cursor
}

func (s *Scanner) restore() {
	s.cursor = s.old
}

func (s *Scanner) skip(accept func(rune) bool) {
	if s.done() {
		return
	}
	for accept(s.char) && !s.done() {
		s.read()
	}
}

func (s *Scanner) done() bool {
	return s.char == utf8.RuneError || s.char == 0
}

func (s *Scanner) read() {
	if s.curr >= len(s.input) {
		s.char = utf8.RuneError
		return
	}
	r, n := utf8.DecodeRune(s.input[s.next:])
	if r == utf8.RuneError {
		s.char = r
		s.next = len(s.input)
		return
	}
	s.char, s.curr, s.next = r, s.next, s.next+n
}

func (s *Scanner) unread() {
	c, z := utf8.DecodeRune(s.input[s.curr:])
	s.char, s.curr, s.next = c, s.curr-z, s.curr
}

func (s *Scanner) peek() rune {
	r, _ := utf8.DecodeRune(s.input[s.next:])
	return r
}

func (s *Scanner) reset() {
	s.str.Reset()
}

func (s *Scanner) write(r rune) {
	s.str.WriteRune(r)
}

func (s *Scanner) literal() string {
	return s.str.String()
}

const (
	dot        = '.'
	pound      = '#'
	colon      = ':'
	space      = ' '
	tab        = '\t'
	nl         = '\n'
	cr         = '\r'
	semicolon  = ';'
	dollar     = '$'
	lparen     = '('
	rparen     = ')'
	lcurly     = '{'
	rcurly     = '}'
	lsquare    = '['
	rsquare    = ']'
	underscore = '_'
	squote     = '\''
	dquote     = '"'
	ampersand  = '&'
	pipe       = '|'
	langle     = '<'
	rangle     = '>'
	equal      = '='
	bang       = '!'
	comma      = ','
	plus       = '+'
	star       = '*'
	minus      = '-'
	percent    = '%'
	slash      = '/'
	caret      = '^'
)

func isTest(r, k rune) bool {
	return (r == lsquare || r == rsquare) && r == k
}

func isCmp(r rune) bool {
	return r == bang || r == equal || r == langle || r == rangle
}

func isLogical(r, k rune) bool {
	return (r == pipe && k == r) || (r == ampersand && r == k)
}

func isGroup(r rune) bool {
	return r == lparen || r == rparen
}

func isPipe(r rune) bool {
	return r == pipe
}

func isRedirect(r, k rune) bool {
	if r == langle || r == rangle || (isPipe(r) && !isPipe(k)) {
		return true
	}
	if r == ampersand && k == rangle {
		return true
	}
	if r == '0' && k == langle {
		return true
	}
	return (r == '1' || r == '2') && k == rangle
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

func isLiteral(r rune) bool {
	return !isAssign(r) && !isEOL(r) && !isBlank(r) && !isComment(r) && !isQuote(r) && !isDollar(r)
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAlpha(r rune) bool {
	return isLetter(r) || isDigit(r) || r == underscore
}

func isAssign(r rune) bool {
	return r == equal
}

func isDollar(r rune) bool {
	return r == dollar
}

func isComment(r rune) bool {
	return r == pound
}

func isEOL(r rune) bool {
	return r == semicolon || isNL(r)
}

func isSpace(r rune) bool {
	return r == space || r == tab
}

func isNL(r rune) bool {
	return r == nl || r == cr
}

func isBlank(r rune) bool {
	return isSpace(r) || isNL(r)
}
