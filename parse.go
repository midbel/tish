package tish

import (
	"fmt"
	"io"
	"strconv"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	infix  map[rune]func(Expr) (Expr, error)
	prefix map[rune]func() (Expr, error)
}

func New(r io.Reader) (*Parser, error) {
	scan, err := Scan(r)
	if err != nil {
		return nil, err
	}
	p := Parser{
		scan:   scan,
		infix:  make(map[rune]func(Expr) (Expr, error)),
		prefix: make(map[rune]func() (Expr, error)),
	}
	p.next()
	p.next()
	return &p, nil
}

func (p *Parser) Parse() (Command, error) {
	if p.done() {
		return nil, io.EOF
	}
	return p.parse()
}

func (p *Parser) parse() (Command, error) {
	p.skip()
	if p.is(Keyword) {
		return p.parseKeyword()
	}
	if p.is(BegList) {
		return p.parseGroup()
	}
	return p.parseCommand()
}

func (p *Parser) parseGroup() (Command, error) {
	if err := p.expect(BegList); err != nil {
		return nil, err
	}
	var list []Command
	for !p.done() && !p.is(EndList) {
		c, err := p.parse()
		if err != nil {
			return nil, err
		}
		list = append(list, c)
		if p.is(EOL) {
			p.next()
		}
	}
	if err := p.expect(EndList); err != nil {
		return nil, err
	}
	if p.is(EOL) {
		p.next()
	}
	return groupCommand(list), nil
}

func (p *Parser) parseKeyword() (Command, error) {
	var parse func() (Command, error)
	switch p.curr.Literal {
	case kwIf, kwElif:
		parse = p.parseIf
	case kwFor:
		parse = p.parseFor
	case kwWhile:
		parse = p.parseWhile
	case kwUntil:
		parse = p.parseUntil
	default:
		return nil, fmt.Errorf("%s unexpected keyword", p.curr.Literal)
	}
	p.next()
	return parse()
}

func (p *Parser) parseAssign() (Command, error) {
	if !p.is(Literal) {
		return nil, fmt.Errorf("parsing assignment! literal expected (%s)", p.curr)
	}
	ident := p.curr.Literal
	p.next()
	if !p.is(Assign) {
		return nil, fmt.Errorf("parsing assignment! literal expected (%s)", p.curr)
	}
	p.next()
	word, err := p.parseWord()
	if err != nil {
		return nil, err
	}
	return assignCommand(ident, word), nil
}

func (p *Parser) parseSingle() (Command, error) {
	var sgl cmdSingle
	for !p.done() && !p.eol() {
		if p.peek.Type != Assign {
			break
		}
		cmd, err := p.parseAssign()
		if err != nil {
			return nil, err
		}
		sgl.export = append(sgl.export, cmd)
		p.skipBlank()
	}
	if p.eoc() {
		return listCommand(sgl.export), nil
	}
	for !p.done() && !p.eoc() {
		if p.is(Assign) {
			p.curr.Literal = "="
			p.curr.Type = Literal
		} else if p.redirect() {
			w, err := p.parseRedirect()
			if err != nil {
				return nil, err
			}
			sgl.redirect = append(sgl.redirect, w)
			continue
		}
		w, err := p.parseWord()
		if err != nil {
			return nil, err
		}
		sgl.words = append(sgl.words, w)
		if !p.eoc() && !p.redirect() {
			p.next()
		}
	}
	return sgl, nil
}

func (p *Parser) parseRedirect() (Word, error) {
	var (
		w Word
		t = p.curr.Type
	)
	if p.is(RedirectOutErr) || p.is(RedirectErrOut) {
		p.next()
		return createRedirect(nil, t), nil
	}
	p.next()
	w, err := p.parseWord()
	if err == nil {
		w = createRedirect(w, t)
	}
	return w, err
}

func (p *Parser) parseCommand() (Command, error) {
	p.skip()
	cmd, err := p.parseSingle()
	if err != nil {
		return nil, err
	}
	if _, ok := cmd.(cmdAssign); ok {
		return cmd, nil
	}
	list := []Command{cmd}
	for !p.done() && p.is(Pipe) {
		p.next()
		c, err := p.parseSingle()
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	if len(list) > 1 {
		cmd = pipeCommand(list)
	}
	var other Command
	switch {
	case p.is(And):
		p.next()
		other, err = p.parseCommand()
		if err == nil {
			cmd = andCommand(cmd, other)
		}
	case p.is(Or):
		p.next()
		other, err = p.parseCommand()
		if err == nil {
			cmd = orCommand(cmd, other)
		}
	case p.is(EOL):
		p.next()
	case p.is(EOF):
	default:
		return nil, fmt.Errorf("unexpected token %s", p.curr)
	}
	return cmd, err
}

func (p *Parser) parseWord() (Word, error) {
	var list []Word
	for !p.eow() {
		var w Word
		switch {
		case p.is(Number):
			n, err := strconv.ParseFloat(p.curr.Literal, 64)
			if err != nil {
				return nil, err
			}
			w = createNumber(n)
		case p.is(Literal):
			w = createLiteral(p.curr.Literal)
		case p.is(Variable):
			w = createIdentifier(p.curr.Literal)
		case p.is(Substitution):
			w = createSubstitution(p.curr.Literal)
		case p.is(Quote):
			w1, err := p.parseQuote()
			if err != nil {
				return nil, err
			}
			w = w1
		case p.is(BegExp):
			w1, err := p.parseExpansion()
			if err != nil {
				return nil, err
			}
			w = w1
		case p.is(BegExpr):
			w1, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			w = w1
		default:
			return nil, fmt.Errorf("parsing word: unexpected token %s", p.curr)
		}
		list = append(list, w)
		p.next()
	}
	return createCombined(list), nil
}

func (p *Parser) parseExpansion() (Word, error) {
	p.next()
	var (
		ident = p.curr.Literal
		word  Word
		err   error
	)
	p.next()
	switch p.curr.Type {
	case ReplaceMatch, ReplaceAll, ReplacePrefix, ReplaceSuffix:
		word, err = p.parseReplaceExpansion(ident)
	case RemSuffix, RemLongSuffix, RemPrefix, RemLongPrefix:
		word, err = p.parseTrimExpansion(ident)
	case Substring:
		word, err = p.parseSubstringExpansion(ident)
	case Lowercase, LowercaseAll, Uppercase, UppercaseAll:
		word, err = p.parseCaseExpansion(ident)
	default:
		return nil, fmt.Errorf("parsing expansion: unrecognized expansion (%s)", p.curr)
	}
	if err != nil {
		return nil, err
	}
	if !p.is(EndExp) {
		return nil, fmt.Errorf("parsing expansion! missing } (%s)", p.curr)
	}
	return word, nil
}

func (p *Parser) parseCaseExpansion(ident string) (Word, error) {
	cs := caser{
		ident: ident,
	}
	switch p.curr.Type {
	case Lowercase, LowercaseAll:
		cs.transformer = getLowercaseTransformer(p.is(LowercaseAll))
	case Uppercase, UppercaseAll:
		cs.transformer = getUppercaseTransformer(p.is(UppercaseAll))
	default:
		return nil, fmt.Errorf("unsupported case transform expansion")
	}
	p.next()
	return cs, nil
}

func (p *Parser) parseReplaceExpansion(ident string) (Word, error) {
	var (
		kind = p.curr.Type
		old  string
		new  string
	)
	p.next()
	if p.is(Literal) {
		old = p.curr.Literal
		p.next()
	}
	if !p.is(ReplaceMatch) {
		return nil, fmt.Errorf("parsing replace! missing / (%s)", p.curr)
	}
	p.next()
	if p.is(Literal) {
		new = p.curr.Literal
		p.next()
	}
	var rep replacer
	switch kind {
	case ReplaceMatch:
		rep = getReplaceOne(old, new)
	case ReplacePrefix:
		rep = getReplacePrefix(old, new)
	case ReplaceSuffix:
		rep = getReplaceSuffix(old, new)
	case ReplaceAll:
		rep = getReplaceAll(old, new)
	default:
		return nil, fmt.Errorf("unsupported replace expansion")
	}
	exp := replace{
		ident:    ident,
		replacer: rep,
	}
	return exp, nil
}

func (p *Parser) parseTrimExpansion(ident string) (Word, error) {
	var (
		kind = p.curr.Type
		str  string
	)
	p.next()
	if !p.is(Literal) {
		return nil, fmt.Errorf("parsing trim! missing string (%s)", p.curr)
	}
	str = p.curr.Literal
	p.next()

	var tr trimmer
	switch kind {
	case RemPrefix, RemLongPrefix:
		tr = getRemovePrefix(str, kind == RemLongPrefix)
	case RemSuffix, RemLongSuffix:
		tr = getRemoveSuffix(str, kind == RemLongSuffix)
	default:
		return nil, fmt.Errorf("unsupported trim expansion")
	}
	exp := trim{
		ident:   ident,
		trimmer: tr,
	}
	return exp, nil
}

func (p *Parser) parseSubstringExpansion(ident string) (Word, error) {
	parse := func(delim rune) (int, error) {
		if p.is(delim) {
			return 0, nil
		}
		var neg bool
		if neg = p.is(BegList); neg {
			p.next()
		}
		if !p.is(Literal) && !p.is(Number) {
			return 0, fmt.Errorf("parse substring! expected literal (%s)", p.curr)
		}
		n, err := strconv.Atoi(p.curr.Literal)
		if err != nil {
			return 0, err
		}
		p.next()
		if neg {
			if !p.is(EndList) {
				return 0, fmt.Errorf("parse substring! expected ) (%s)", p.curr)
			}
			p.next()
		}
		return n, err
	}
	p.next()
	var (
		err error
		exp = substring{
			ident: ident,
		}
	)
	exp.offset, err = parse(Substring)
	if err != nil {
		return nil, err
	}
	switch {
	case p.is(Substring):
		p.next()
	case p.is(EndExp):
		return exp, nil
	default:
		return nil, fmt.Errorf("parsing substring! missing : (%s)", p.curr)
	}
	exp.length, err = parse(EndExp)
	if err != nil {
		return nil, err
	}
	return exp, nil
}

func (p *Parser) parseQuote() (Word, error) {
	p.next()
	var list []Word
	for !p.is(Quote) && !p.done() {
		var w Word
		switch {
		case p.is(Literal):
			w = createLiteral(p.curr.Literal)
		case p.is(Variable):
			w = createIdentifier(p.curr.Literal)
		case p.is(Substitution):
			w = createSubstitution(p.curr.Literal)
		default:
			return nil, fmt.Errorf("parsing quote: unexpected token %s", p.curr)
		}
		list = append(list, w)
		p.next()
	}
	if !p.is(Quote) {
		return nil, fmt.Errorf("parsing quote: missing quote (%s)", p.curr)
	}
	return createCombined(list), nil
}

func (p *Parser) parseBody(until func() bool) (Command, error) {
	var list cmdList
	for !p.done() {
		if until() {
			break
		}
		c, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		list.commands = append(list.commands, c)
	}
	if len(list.commands) == 1 {
		return list.commands[0], nil
	}
	return list, nil
}

func (p *Parser) parseIf() (Command, error) {
	p.skipBlank()
	var (
		cmd cmdIf
		err error
	)
	if p.is(BegTest) {
		cmd.test, err = p.parseTest()
		p.skip()
	} else {
		cmd.test, err = p.parseCommand()
	}
	if err != nil {
		return nil, err
	}
	p.skipBlank()
	if !p.is(Keyword) && p.curr.Literal != kwThen {
		return nil, fmt.Errorf("parsing if: missing then (%s)", p.curr)
	}
	p.next()
	p.skipBlank()

	cmd.csq, err = p.parseBody(func() bool {
		return p.is(Keyword) && (p.curr.Literal == kwFi || p.curr.Literal == kwElif || p.curr.Literal == kwElse)
	})
	if err != nil {
		return nil, err
	}
	switch {
	case p.is(Keyword) && p.curr.Literal == kwElif:
		cmd.alt, err = p.parseKeyword()
	case p.is(Keyword) && p.curr.Literal == kwElse:
		cmd.alt, err = p.parseBody(func() bool {
			return p.is(Keyword) && p.curr.Literal == kwFi
		})
		if !p.is(Keyword) && p.curr.Literal != kwFi {
			err = fmt.Errorf("parsing if: unexpected token (%s)", p.curr)
		}
	case p.is(Keyword) && p.curr.Literal == kwFi:
	default:
		return nil, fmt.Errorf("parsing if: unexpected token (%s)", p.curr)
	}
	if err != nil {
		return nil, err
	}
	p.next()
	p.skip()
	return cmd, nil
}

func (p *Parser) parseFor() (Command, error) {
	p.skipBlank()

	var (
		cmd cmdFor
		err error
	)

	if !p.is(Literal) {
		return nil, fmt.Errorf("parsing for: expected literal (%s)", p.curr)
	}
	cmd.ident = p.curr.Literal
	p.next()
	p.skipBlank()
	if !p.is(Keyword) && p.curr.Literal != kwIn {
		return nil, fmt.Errorf("parsing for: missing in (%s)", p.curr)
	}
	p.next()
	p.skipBlank()
	if cmd.iter, err = p.parseCommand(); err != nil {
		return nil, err
	}
	if !p.is(Keyword) && p.curr.Literal != kwDo {
		return nil, fmt.Errorf("parsing for: missing do (%s)", p.curr)
	}
	p.next()
	p.skipBlank()

	cmd.body, err = p.parseBody(func() bool {
		return p.is(Keyword) && p.curr.Literal == kwDone
	})
	if err != nil {
		return nil, err
	}
	if !p.is(Keyword) && p.curr.Literal != kwDone {
		return nil, fmt.Errorf("parsing for: missing done (%s)", p.curr)
	}
	p.next()
	p.skip()
	return cmd, nil
}

func (p *Parser) parseWhile() (Command, error) {
	p.skipBlank()

	var (
		cmd cmdWhile
		err error
	)
	if cmd.iter, err = p.parseCommand(); err != nil {
		return nil, err
	}
	if !p.is(Keyword) && p.curr.Literal != kwDo {
		return nil, fmt.Errorf("parsing while: missing do (%s)", p.curr)
	}
	p.next()
	p.skipBlank()

	cmd.body, err = p.parseBody(func() bool {
		return p.is(Keyword) && p.curr.Literal == kwDone
	})
	if err != nil {
		return nil, err
	}
	if !p.is(Keyword) && p.curr.Literal != kwDone {
		return nil, fmt.Errorf("parsing while: missing done (%s)", p.curr)
	}
	p.next()
	p.skip()
	return cmd, nil
}

func (p *Parser) parseUntil() (Command, error) {
	p.skipBlank()

	var (
		cmd cmdUntil
		err error
	)
	if cmd.iter, err = p.parseCommand(); err != nil {
		return nil, err
	}
	if !p.is(Keyword) && p.curr.Literal != kwDo {
		return nil, fmt.Errorf("parsing until: missing do (%s)", p.curr)
	}
	p.next()
	p.skipBlank()

	cmd.body, err = p.parseBody(func() bool {
		return p.is(Keyword) && p.curr.Literal == kwDone
	})
	if err != nil {
		return nil, err
	}
	if !p.is(Keyword) && p.curr.Literal != kwDone {
		return nil, fmt.Errorf("parsing until: missing done (%s)", p.curr)
	}
	p.next()
	p.skip()
	return cmd, nil
}

func (p *Parser) parseExpr() (Word, error) {
	p.next()
	p.registerExprFunc()
	e, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	if !p.is(EndExpr) {
		return nil, fmt.Errorf("parsing expr: missing closing expr (%s)", p.curr)
	}
	return createExpr(e), nil
}

func (p *Parser) parseTest() (Command, error) {
	p.next()
	p.registerTestFunc()
	e, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	if !p.is(EndTest) {
		return nil, fmt.Errorf("parsing test: missing closing test (%s)", p.curr)
	}
	p.next()
	return testCommand(e), nil
}

func (p *Parser) parseExpression(pow int) (Expr, error) {
	fn, ok := p.prefix[p.curr.Type]
	if !ok {
		return nil, fmt.Errorf("%s: prefix operator not supported", p.curr)
	}
	left, err := fn()
	if err != nil {
		return nil, err
	}
	for (!p.is(EndTest) && !p.is(EndExpr)) && pow < powers.Get(p.curr.Type) {
		fn, ok := p.infix[p.curr.Type]
		if !ok {
			return nil, fmt.Errorf("%s: infix operator not supported", p.curr)
		}
		left, err = fn(left)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) parseGroupExpr() (Expr, error) {
	p.next()
	e, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	if !p.is(EndList) {
		return nil, fmt.Errorf("missing closing parens (%s)", p.curr)
	}
	p.next()
	return e, nil
}

func (p *Parser) parseNotExpr() (Expr, error) {
	p.next()
	e, err := p.parseExpression(powUnary)
	if err != nil {
		return nil, err
	}
	e = unaryExpr{
		op:   "!",
		word: e,
	}
	return e, nil
}

func (p *Parser) parseRevExpr() (Expr, error) {
	p.next()
	e, err := p.parseExpression(powUnary)
	if err != nil {
		return nil, err
	}
	e = unaryExpr{
		op:   "-",
		word: e,
	}
	return e, nil
}

func (p *Parser) parseUnaryExpr() (Expr, error) {
	op := p.curr.Literal
	p.next()
	e, err := p.parseExpression(powUnary)
	if err != nil {
		return nil, err
	}
	e = unaryExpr{
		op:   op,
		word: e,
	}
	return e, nil
}

func (p *Parser) parseWordExpr() (Expr, error) {
	w, err := p.parseWord()
	if err != nil {
		return nil, err
	}
	e, ok := w.(Expr)
	if !ok {
		return nil, fmt.Errorf("word is not an expression")
	}
	return e, nil
}

func (p *Parser) parseBinaryExpr(left Expr) (Expr, error) {
	var (
		op  string
		pow = powers.Get(p.curr.Type)
	)
	switch {
	case p.is(Option):
		op = p.curr.Literal
	case p.is(And):
		op = "&&"
	case p.is(Or):
		op = "||"
	case p.is(Eq):
		op = "=="
	case p.is(Ne):
		op = "!="
	case p.is(Lt):
		op = "<"
	case p.is(Le):
		op = "<="
	case p.is(Gt):
		op = ">"
	case p.is(Ge):
		op = ">="
	case p.is(Add):
		op = "+"
	case p.is(Sub):
		op = "-"
	case p.is(Mul):
		op = "*"
	case p.is(Div):
		op = "/"
	case p.is(Mod):
		op = "%"
	case p.is(Lshift):
		op = "<<"
	case p.is(Rshift):
		op = ">>"
	case p.is(Band):
		op = "&"
	case p.is(Bor):
		op = "|"
	default:
		return nil, fmt.Errorf("%s: unsupported binary operator", p.curr)
	}
	p.next()
	e := binaryExpr{
		op:   op,
		left: left,
	}
	right, err := p.parseExpression(pow)
	if err != nil {
		return nil, err
	}
	e.right = right
	return e, nil
}

func (p *Parser) expect(k rune) error {
	if !p.is(k) {
		return fmt.Errorf("unexpected token %s", p.curr)
	}
	p.next()
	return nil
}

// end of command
func (p *Parser) eoc() bool {
	return p.eol() || p.is(Pipe) || p.is(And) || p.is(Or)
}

// end of line
func (p *Parser) eol() bool {
	return p.is(EOL) || p.is(Comment) || p.done()
}

// end of word
func (p *Parser) eow() bool {
	switch p.curr.Type {
	case RedirectIn:
	case RedirectOut:
	case RedirectErr:
	case RedirectErrOut:
	case RedirectOutErr:
	case RedirectBoth:
	case AppendOut:
	case AppendErr:
	case AppendBoth:
	case Assign:
	case RemSuffix:
	case RemLongSuffix:
	case RemPrefix:
	case RemLongPrefix:
	case ReplaceMatch:
	case ReplaceAll:
	case ReplacePrefix:
	case ReplaceSuffix:
	case Substring:
	case Blank:
	case EndTest:
	case EndList:
	case EndExp:
	case EndExpr:
	case Add:
	case Sub:
	case Mul:
	case Div:
	case Mod:
	case Band:
	case Bor:
	case Lshift:
	case Rshift:
	default:
		return p.eoc()
	}
	return true
}

func (p *Parser) redirect() bool {
	switch p.curr.Type {
	case RedirectIn:
	case RedirectOut:
	case RedirectErr:
	case RedirectBoth:
	case RedirectErrOut:
	case RedirectOutErr:
	case AppendOut:
	case AppendErr:
	case AppendBoth:
	default:
		return false
	}
	return true
}

func (p *Parser) is(kind rune) bool {
	return p.curr.Type == kind
}

func (p *Parser) skip() {
	for (p.is(Comment) || p.is(EOL)) && !p.done() {
		p.next()
	}
}

func (p *Parser) skipBlank() {
	for !p.done() && p.is(Blank) {
		p.next()
	}
}

func (p *Parser) done() bool {
	return p.curr.Type == EOF
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) registerTestFunc() {
	p.clearFunc()
	p.registerPrefix(Option, p.parseUnaryExpr)
	p.registerPrefix(BegList, p.parseGroupExpr)
	p.registerPrefix(Not, p.parseNotExpr)
	p.registerPrefix(Literal, p.parseWordExpr)
	p.registerPrefix(Variable, p.parseWordExpr)
	p.registerPrefix(Quote, p.parseWordExpr)
	p.registerInfix(Option, p.parseBinaryExpr)
	p.registerInfix(And, p.parseBinaryExpr)
	p.registerInfix(Or, p.parseBinaryExpr)
	p.registerInfix(Eq, p.parseBinaryExpr)
	p.registerInfix(Ne, p.parseBinaryExpr)
	p.registerInfix(Lt, p.parseBinaryExpr)
	p.registerInfix(Le, p.parseBinaryExpr)
	p.registerInfix(Gt, p.parseBinaryExpr)
	p.registerInfix(Ge, p.parseBinaryExpr)
}

func (p *Parser) registerExprFunc() {
	p.clearFunc()
	p.registerPrefix(BegList, p.parseGroupExpr)
	p.registerPrefix(Not, p.parseNotExpr)
	p.registerPrefix(Sub, p.parseRevExpr)
	p.registerPrefix(Literal, p.parseWordExpr)
	p.registerPrefix(Number, p.parseWordExpr)
	p.registerPrefix(Variable, p.parseWordExpr)
	p.registerInfix(And, p.parseBinaryExpr)
	p.registerInfix(Or, p.parseBinaryExpr)
	p.registerInfix(Eq, p.parseBinaryExpr)
	p.registerInfix(Ne, p.parseBinaryExpr)
	p.registerInfix(Add, p.parseBinaryExpr)
	p.registerInfix(Sub, p.parseBinaryExpr)
	p.registerInfix(Mul, p.parseBinaryExpr)
	p.registerInfix(Div, p.parseBinaryExpr)
	p.registerInfix(Mod, p.parseBinaryExpr)
	p.registerInfix(Band, p.parseBinaryExpr)
	p.registerInfix(Bor, p.parseBinaryExpr)
}

func (p *Parser) clearFunc() {
	clear(p.infix)
	clear(p.prefix)
}

func (p *Parser) registerInfix(r rune, fn func(Expr) (Expr, error)) {
	p.infix[r] = fn
}

func (p *Parser) registerPrefix(r rune, fn func() (Expr, error)) {
	p.prefix[r] = fn
}

const (
	powLowest int = iota
	powOr
	powAnd
	powEq
	powCmp
	powBin
	powShift
	powAdd
	powMul
	powUnary
)

type binding map[rune]int

var powers = binding{
	And:    powAnd,
	Or:     powOr,
	Eq:     powEq,
	Ne:     powEq,
	Lt:     powCmp,
	Le:     powCmp,
	Gt:     powCmp,
	Ge:     powCmp,
	Option: powCmp,
	Add:    powAdd,
	Sub:    powAdd,
	Mul:    powMul,
	Div:    powMul,
	Mod:    powMul,
	Band:   powBin,
	Bor:    powBin,
	Lshift: powShift,
	Rshift: powShift,
}

func (b binding) Get(r rune) int {
	return b[r]
}

const (
	kwIf    = "if"
	kwThen  = "then"
	kwFi    = "fi"
	kwElif  = "elif"
	kwElse  = "else"
	kwFor   = "for"
	kwDo    = "do"
	kwDone  = "done"
	kwWhile = "while"
	kwUntil = "until"
	kwIn    = "in"
)

func isKeyword(str string) bool {
	switch str {
	case kwIf:
	case kwThen:
	case kwElif:
	case kwElse:
	case kwFi:
	case kwFor:
	case kwDo:
	case kwDone:
	case kwWhile:
	case kwUntil:
	case kwIn:
	default:
		return false
	}
	return true
}
