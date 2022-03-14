package tish

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	quoted bool
	prefix map[rune]func() (Expr, error)
	infix  map[rune]func(Expr) (Expr, error)

	unary  map[rune]func() (Expander, error)
	binary map[rune]func(Expander) (Expander, error)

	loop int
}

func Parse(str string) (Executer, error) {
	p := NewParser(strings.NewReader(str))
	return p.Parse()
}

func NewParser(r io.Reader) *Parser {
	var p Parser
	p.scan = Scan(r)

	p.prefix = map[rune]func() (Expr, error){
		BegMath:  p.parseUnary,
		Numeric:  p.parseUnary,
		Variable: p.parseUnary,
		Sub:      p.parseUnary,
		Inc:      p.parseUnary,
		Dec:      p.parseUnary,
		BitNot:   p.parseUnary,
		Not:      p.parseUnary,
	}
	p.infix = map[rune]func(Expr) (Expr, error){
		Add:        p.parseBinary,
		Sub:        p.parseBinary,
		Mul:        p.parseBinary,
		Div:        p.parseBinary,
		Pow:        p.parseBinary,
		LeftShift:  p.parseBinary,
		RightShift: p.parseBinary,
		BitAnd:     p.parseBinary,
		BitOr:      p.parseBinary,
		BitXor:     p.parseBinary,
		BitNot:     p.parseBinary,
		Eq:         p.parseBinary,
		Ne:         p.parseBinary,
		Lt:         p.parseBinary,
		Le:         p.parseBinary,
		Gt:         p.parseBinary,
		Ge:         p.parseBinary,
		And:        p.parseBinary,
		Or:         p.parseBinary,
		Cond:       p.parseTernary,
		Assign:     p.parseAssign,
	}

	p.binary = map[rune]func(Expander) (Expander, error){
		Eq:        p.parseBinaryTest,
		Ne:        p.parseBinaryTest,
		Lt:        p.parseBinaryTest,
		Le:        p.parseBinaryTest,
		Gt:        p.parseBinaryTest,
		Ge:        p.parseBinaryTest,
		And:       p.parseBinaryTest,
		Or:        p.parseBinaryTest,
		NewerThan: p.parseBinaryTest,
		OlderThan: p.parseBinaryTest,
		SameFile:  p.parseBinaryTest,
	}
	p.unary = map[rune]func() (Expander, error){
		Not:         p.parseUnaryTest,
		BegMath:     p.parseUnaryTest,
		FileExists:  p.parseUnaryTest,
		FileRead:    p.parseUnaryTest,
		FileLink:    p.parseUnaryTest,
		FileDir:     p.parseUnaryTest,
		FileWrite:   p.parseUnaryTest,
		FileSize:    p.parseUnaryTest,
		FileRegular: p.parseUnaryTest,
		FileExec:    p.parseUnaryTest,
		StrNotEmpty: p.parseUnaryTest,
		StrEmpty:    p.parseUnaryTest,
		Literal:     p.parseUnaryTest,
		Variable:    p.parseUnaryTest,
		Quote:       p.parseUnaryTest,
	}

	p.next()
	p.next()

	return &p
}

func (p *Parser) Parse() (Executer, error) {
	if p.done() {
		return nil, io.EOF
	}
	ex, err := p.parse()
	if err != nil {
		return nil, err
	}
	switch p.curr.Type {
	case List, Comment, EOF:
		p.next()
	default:
		return nil, p.unexpected()
	}
	return ex, nil
}

func (p *Parser) parse() (Executer, error) {
	switch {
	case p.peek.Type == Assign:
		return p.parseAssignment()
	case p.curr.Type == Keyword:
		return p.parseKeyword()
	case p.curr.Type == BegTest:
		return p.parseTest()
	case p.curr.Type == BegSub:
		return p.parseSubshell()
	default:
	}
	ex, err := p.parseSimple()
	if err != nil {
		return nil, err
	}
	for {
		switch p.curr.Type {
		case And:
			return p.parseAnd(ex)
		case Or:
			return p.parseOr(ex)
		case Pipe, PipeBoth:
			ex, err = p.parsePipe(ex)
			if err != nil {
				return nil, err
			}
		default:
			return ex, nil
		}
	}
}

func (p *Parser) parseSubshell() (Executer, error) {
	p.next()
	var list ExecSubshell
	for !p.done() && p.curr.Type != EndSub {
		if p.curr.Type == EndSub {
			break
		}
		if p.curr.Type == List {
			p.next()
		}
		p.skipBlank()
		x, err := p.parse()
		if err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	if p.curr.Type != EndSub {
		return nil, p.unexpected()
	}
	p.next()
	return list.Executer(), nil
}

func (p *Parser) parseTest() (Executer, error) {
	p.next()
	ex, err := p.parseTester(bindLowest)
	if err != nil {
		return nil, err
	}
	if p.curr.Type != EndTest {
		return nil, p.unexpected()
	}
	p.next()

	var test ExecTest
	if x, ok := ex.(Tester); ok {
		test.Tester = x
	} else {
		test.Tester = SingleTest{
			Expander: ex,
		}
	}
	return test, err
}

func (p *Parser) parseTester(pow bind) (Expander, error) {
	parse, ok := p.unary[p.curr.Type]
	if !ok {
		return nil, p.unexpected()
	}
	left, err := parse()
	if err != nil {
		return nil, err
	}
	for (p.curr.Type != EndMath && p.curr.Type != EndTest) && pow < bindPower(p.curr) {
		parse, ok := p.binary[p.curr.Type]
		if !ok {
			return nil, p.unexpected()
		}
		left, err = parse(left)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) parseUnaryTest() (Expander, error) {
	var (
		ex  Expander
		err error
	)
	switch op := p.curr.Type; op {
	case Variable:
		ex, err = p.parseVariable()
	case Literal:
		ex, err = p.parseLiteral()
	case Quote:
		ex, err = p.parseQuote()
		p.skipBlank()
	case BegMath:
		p.next()
		ex, err = p.parseTester(bindLowest)
		if err != nil {
			return nil, err
		}
		if p.curr.Type != EndMath {
			err = p.unexpected()
			break
		}
		p.next()
	case Not, FileExists, FileRead, FileWrite, FileExec, FileSize, FileLink, FileDir, FileRegular, StrEmpty, StrNotEmpty:
		p.next()
		ex, err = p.parseTester(bindPrefix)
		if err != nil {
			break
		}
		ex = UnaryTest{
			Op:    op,
			Right: ex,
		}
	default:
		err = p.unexpected()
	}
	if err != nil {
		return nil, err
	}
	return ex, nil
}

func (p *Parser) parseBinaryTest(left Expander) (Expander, error) {
	b := BinaryTest{
		Left: left,
		Op:   p.curr.Type,
	}
	w := bindPower(p.curr)
	p.next()

	right, err := p.parseTester(w)
	if err == nil {
		b.Right = right
	}
	return b, err
}

func (p *Parser) parseSimple() (Executer, error) {
	var (
		ex   ExpandList
		dirs []ExpandRedirect
	)
	for {
		switch p.curr.Type {
		case Literal, Quote, Variable, BegExp, BegBrace, BegSub, BegMath:
			next, err := p.parseWords()
			if err != nil {
				return nil, err
			}
			ex.List = append(ex.List, next)
		case RedirectIn, RedirectOut, RedirectErr, RedirectBoth, AppendOut, AppendBoth:
			next, err := p.parseRedirection()
			if err != nil {
				return nil, err
			}
			dirs = append(dirs, next)
		default:
			sg := createSimple(ex)
			sg.Redirect = append(sg.Redirect, dirs...)
			return sg, nil
		}
	}
}

func (p *Parser) parseRedirection() (ExpandRedirect, error) {
	kind := p.curr.Type
	p.next()
	e, err := p.parseWords()
	if err != nil {
		return ExpandRedirect{}, err
	}
	return createRedirect(e, kind), nil
}

func (p *Parser) parseAssignment() (Executer, error) {
	if p.curr.Type != Literal {
		return nil, p.unexpected()
	}
	var (
		ident = p.curr.Literal
		list  ExpandList
	)
	p.next()
	if p.curr.Type != Assign {
		return nil, p.unexpected()
	}
	p.next()
	for !p.done() {
		if p.curr.IsSequence() {
			break
		}
		w, err := p.parseWords()
		if err != nil {
			return nil, err
		}
		list.List = append(list.List, w)
	}
	return createAssign(ident, list), nil
}

func (p *Parser) parsePipe(left Executer) (Executer, error) {
	var list []pipeitem
	list = append(list, createPipeItem(left, p.curr.Type == PipeBoth))
	for !p.done() {
		if p.curr.Type != Pipe && p.curr.Type != PipeBoth {
			break
		}
		var (
			both = p.curr.Type == PipeBoth
			ex   Executer
			err  error
		)
		p.next()
		if ex, err = p.parseSimple(); err != nil {
			return nil, err
		}
		list = append(list, createPipeItem(ex, both))
	}
	return createPipe(list), nil
}

func (p *Parser) parseAnd(left Executer) (Executer, error) {
	p.next()
	right, err := p.parse()
	if err != nil {
		return nil, err
	}
	return createAnd(left, right), nil
}

func (p *Parser) parseOr(left Executer) (Executer, error) {
	p.next()
	right, err := p.parse()
	if err != nil {
		return nil, err
	}
	return createOr(left, right), nil
}

func (p *Parser) parseKeyword() (Executer, error) {
	var (
		ex  Executer
		err error
	)
	switch p.curr.Literal {
	case kwBreak:
		if !p.inLoop() {
			return nil, p.unexpected()
		}
		ex = ExecBreak{}
		p.next()
	case kwContinue:
		if !p.inLoop() {
			return nil, p.unexpected()
		}
		ex = ExecContinue{}
		p.next()
	case kwFor:
		ex, err = p.parseFor()
	case kwWhile:
		ex, err = p.parseWhile()
	case kwUntil:
		ex, err = p.parseUntil()
	case kwIf:
		ex, err = p.parseIf()
	case kwCase:
		ex, err = p.parseCase()
	default:
		err = p.unexpected()
	}
	return ex, err
}

func (p *Parser) parseWhile() (Executer, error) {
	p.enterLoop()
	defer p.leaveLoop()

	p.next()
	p.skipBlank()
	var (
		ex  ExecWhile
		err error
	)
	if ex.Cond, err = p.parse(); err != nil {
		return nil, err
	}
	if p.curr.Type != List {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type != Keyword || p.curr.Literal != kwDo {
		return nil, p.unexpected()
	}
	ex.Body, err = p.parseBody(func(kw string) bool { return kw == kwElse || kw == kwDone })
	if err != nil {
		return nil, err
	}
	if p.curr.Type == Keyword && p.curr.Literal == kwElse {
		ex.Alt, err = p.parseBody(func(kw string) bool { return kw == kwDone })
		if err != nil {
			return nil, err
		}
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseUntil() (Executer, error) {
	p.enterLoop()
	defer p.leaveLoop()

	p.next()
	p.skipBlank()
	var (
		ex  ExecUntil
		err error
	)
	if ex.Cond, err = p.parse(); err != nil {
		return nil, err
	}
	if p.curr.Type != List {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type != Keyword || p.curr.Literal != kwDo {
		return nil, p.unexpected()
	}
	ex.Body, err = p.parseBody(func(kw string) bool { return kw == kwElse || kw == kwDone })
	if err != nil {
		return nil, err
	}
	if p.curr.Type == Keyword && p.curr.Literal == kwElse {
		ex.Alt, err = p.parseBody(func(kw string) bool { return kw == kwDone })
		if err != nil {
			return nil, err
		}
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseIf() (Executer, error) {
	p.next()
	p.skipBlank()

	var (
		ex  ExecIf
		err error
	)
	if ex.Cond, err = p.parse(); err != nil {
		return nil, err
	}
	if p.curr.Type != List {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type != Keyword || p.curr.Literal != kwThen {
		return nil, p.unexpected()
	}
	ex.Csq, err = p.parseBody(func(kw string) bool { return kw == kwElse || kw == kwFi })
	if err != nil {
		return nil, err
	}
	if p.curr.Type == Keyword && p.curr.Literal == kwElse {
		p.next()
		ex.Alt, err = p.parse()
		if err != nil {
			return nil, err
		}
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseClause() (ExecClause, error) {
	var c ExecClause
	if p.curr.Literal == "*" && p.peek.Type != EndSub {
		return c, p.unexpected()
	}
	for !p.done() && p.curr.Type != EndSub {
		var word Expander
		switch p.curr.Type {
		case Literal:
			word, _ = p.parseLiteral()
		case Variable:
			word, _ = p.parseVariable()
		default:
			return c, p.unexpected()
		}
		c.List = append(c.List, word)
		switch p.curr.Type {
		case Comma:
			p.next()
		case EndSub:
		default:
			return c, p.unexpected()
		}
	}
	if p.curr.Type != EndSub {
		return c, p.unexpected()
	}
	p.next()
	var list ExecList
	for !p.done() {
		if p.curr.Type == List {
			p.next()
			p.skipBlank()
		}
		if p.peek.Type == Comma || p.peek.Type == EndSub || (p.curr.Type == Keyword && p.curr.Literal == kwEsac) {
			break
		}
		p.skipBlank()
		x, err := p.parse()
		if err != nil {
			return c, err
		}
		list = append(list, x)
	}
	c.Body = list.Executer()
	return c, nil
}

func (p *Parser) parseCase() (Executer, error) {
	p.next()
	var ex ExecCase
	switch p.curr.Type {
	case Literal:
		ex.Word, _ = p.parseLiteral()
	case Variable:
		ex.Word, _ = p.parseVariable()
	default:
		return nil, p.unexpected()
	}
	p.next()
	p.skipBlank()
	if p.curr.Type != Keyword && p.curr.Literal != kwIn {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type != List {
		return nil, p.unexpected()
	}
	p.next()
	for p.curr.Type != Keyword && p.curr.Literal != kwEsac {
		fallback := p.curr.Type == Literal && p.curr.Literal == "*"
		c, err := p.parseClause()
		if err != nil {
			return nil, err
		}
		if !fallback {
			ex.List = append(ex.List, c)
		} else {
			ex.Default = c.Body
		}
	}
	if p.curr.Type != Keyword && p.curr.Literal != kwEsac {
		return nil, p.unexpected()
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseFor() (Executer, error) {
	p.enterLoop()
	defer p.leaveLoop()

	p.next()
	p.skipBlank()
	if p.curr.Type != Literal {
		return nil, p.unexpected()
	}
	ex := ExecFor{
		Ident: p.curr.Literal,
	}
	p.next()
	p.skipBlank()
	if p.curr.Type != Keyword || p.curr.Literal != kwIn {
		return nil, p.unexpected()
	}
	p.next()
	p.skipBlank()
	for !p.done() && p.curr.Type != List {
		e, err := p.parseWords()
		if err != nil {
			return nil, err
		}
		ex.List = append(ex.List, e)
	}
	if p.curr.Type != List {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type != Keyword && p.curr.Literal != kwDo {
		return nil, p.unexpected()
	}
	var err error
	ex.Body, err = p.parseBody(func(kw string) bool { return kw == kwElse || kw == kwDone })
	if err != nil {
		return nil, err
	}
	if p.curr.Type == Keyword && p.curr.Literal == kwElse {
		ex.Alt, err = p.parseBody(func(kw string) bool { return kw == kwDone })
		if err != nil {
			return nil, err
		}
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseBody(stop func(kw string) bool) (Executer, error) {
	var list ExecList
	p.next()
	for !p.done() && !stop(p.curr.Literal) {
		switch p.curr.Type {
		case Blank:
			p.skipBlank()
			continue
		case List:
			p.next()
			continue
		default:
		}
		e, err := p.parse()
		if err != nil {
			return nil, err
		}
		list = append(list, e)
		switch p.curr.Type {
		case List, Comment:
			p.next()
		case Keyword:
			if !stop(p.curr.Literal) {
				return nil, p.unexpected()
			}
		default:
			return nil, p.unexpected()
		}
	}
	if p.curr.Type != Keyword || !stop(p.curr.Literal) {
		return nil, p.unexpected()
	}
	return list.Executer(), nil
}

func (p *Parser) parseWords() (Expander, error) {
	var list ExpandMulti
	for !p.done() {
		if p.curr.Eow() {
			if !p.curr.IsSequence() {
				p.next()
			}
			break
		}
		var (
			next Expander
			err  error
		)
		switch p.curr.Type {
		case Literal:
			next, err = p.parseLiteral()
		case Variable:
			next, err = p.parseVariable()
		case Quote:
			next, err = p.parseQuote()
		case BegExp:
			next, err = p.parseExpansion()
		case BegSub:
			next, err = p.parseSubstitution()
		case BegMath:
			next, err = p.parseArithmetic()
		case BegBrace:
			next, err = p.parseBraces(list.Pop())
		default:
			err = p.unexpected()
		}
		if err != nil {
			return nil, err
		}
		list.List = append(list.List, next)
	}
	return list.Expander(), nil
}

func (p *Parser) parseArithmetic() (Expander, error) {
	p.next()
	var list ExpandMath
	list.Quoted = p.quoted
	for !p.done() && p.curr.Type != EndMath {
		next, err := p.parseExpression(bindLowest)
		if err != nil {
			return nil, err
		}
		switch p.curr.Type {
		case List:
			p.next()
		case EndMath:
		default:
			return nil, p.unexpected()
		}
		list.List = append(list.List, next)
	}
	if p.curr.Type != EndMath {
		return nil, p.unexpected()
	}
	p.next()
	return list, nil
}

func (p *Parser) parseExpression(pow bind) (Expr, error) {
	fn, ok := p.prefix[p.curr.Type]
	if !ok {
		return nil, p.unexpected()
	}
	left, err := fn()
	if err != nil {
		return nil, err
	}
	for (p.curr.Type != EndMath && p.curr.Type != List) && pow < bindPower(p.curr) {
		fn, ok := p.infix[p.curr.Type]
		if !ok {
			return nil, p.unexpected()
		}
		left, err = fn(left)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) parseUnary() (Expr, error) {
	var (
		ex  Expr
		err error
	)
	switch p.curr.Type {
	case Sub, Inc, Dec, Not, BitNot:
		op := p.curr.Type
		p.next()
		ex, err = p.parseExpression(bindPrefix)
		if err != nil {
			break
		}
		ex = createUnary(ex, op)
	case BegMath:
		p.next()
		ex, err = p.parseExpression(bindLowest)
		if err != nil {
			break
		}
		if p.curr.Type != EndMath {
			err = p.unexpected()
			break
		}
		p.next()
	case Numeric:
		ex = createNumber(p.curr.Literal)
		p.next()
	case Variable:
		ex = createVariable(p.curr.Literal, false)
		p.next()
	default:
		return nil, p.unexpected()
	}
	return ex, err
}

func (p *Parser) parseBinary(left Expr) (Expr, error) {
	b := Binary{
		Left: left,
		Op:   p.curr.Type,
	}
	w := bindPower(p.curr)
	p.next()

	right, err := p.parseExpression(w)
	if err == nil {
		b.Right = right
	}
	return b, err
}

func (p *Parser) parseAssign(left Expr) (Expr, error) {
	var as Assignment
	switch v := left.(type) {
	case ExpandVar:
		as.Ident = v.Ident
	default:
		return nil, p.unexpected()
	}
	p.next()

	expr, err := p.parseExpression(bindLowest)
	if err != nil {
		return nil, err
	}
	as.Expr = expr
	return as, nil
}

func (p *Parser) parseTernary(left Expr) (Expr, error) {
	p.next()
	ter := Ternary{
		Cond: left,
	}
	left, err := p.parseExpression(bindTernary)
	if err != nil {
		return nil, err
	}
	ter.Left = left
	if p.curr.Type != Alt {
		return nil, p.unexpected()
	}
	p.next()
	right, err := p.parseExpression(bindLowest)
	if err != nil {
		return nil, err
	}
	ter.Right = right
	return ter, nil
}

func (p *Parser) parseSubstitution() (Expander, error) {
	var ex ExpandSub
	ex.Quoted = p.quoted
	p.next()
	for !p.done() && p.curr.Type != EndSub {
		next, err := p.parse()
		if err != nil {
			return nil, err
		}
		ex.List = append(ex.List, next)
	}
	if p.curr.Type != EndSub {
		return nil, p.unexpected()
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseQuote() (Expander, error) {
	p.enterQuote()
	p.next()

	var list ExpandMulti
	for !p.done() && p.curr.Type != Quote {
		var (
			next Expander
			err  error
		)
		switch p.curr.Type {
		case Literal:
			next, err = p.parseLiteral()
		case Variable:
			next, err = p.parseVariable()
		case BegExp:
			next, err = p.parseExpansion()
		case BegSub:
			next, err = p.parseSubstitution()
		case BegMath:
			next, err = p.parseArithmetic()
		default:
			err = p.unexpected()
		}
		if err != nil {
			return nil, err
		}
		list.List = append(list.List, next)
	}
	if p.curr.Type != Quote {
		return nil, p.unexpected()
	}
	p.leaveQuote()
	p.next()
	return list.Expander(), nil
}

func (p *Parser) parseLiteral() (ExpandWord, error) {
	ex := createWord(p.curr.Literal, p.quoted)
	p.next()
	return ex, nil
}

func (p *Parser) parseBraces(prefix Expander) (Expander, error) {
	p.next()
	if p.peek.Type == Range {
		return p.parseRangeBraces(prefix)
	}
	return p.parseListBraces(prefix)
}

func (p *Parser) parseWordsInBraces() (Expander, error) {
	var list ExpandList
	for !p.done() {
		if p.curr.Type == Seq || p.curr.Type == EndBrace {
			break
		}
		var (
			next Expander
			err  error
		)
		switch p.curr.Type {
		case Literal:
			next, err = p.parseLiteral()
		case BegBrace:
			next, err = p.parseBraces(list.Pop())
		default:
			err = p.unexpected()
		}
		if err != nil {
			return nil, err
		}
		list.List = append(list.List, next)
	}
	return list, nil
}

func (p *Parser) parseListBraces(prefix Expander) (Expander, error) {
	ex := ExpandListBrace{
		Prefix: prefix,
	}
	for !p.done() {
		if p.curr.Type == EndBrace {
			break
		}
		x, err := p.parseWordsInBraces()
		if err != nil {
			return nil, err
		}
		ex.Words = append(ex.Words, x)
		switch p.curr.Type {
		case Seq:
			p.next()
		case EndBrace:
		default:
			return nil, p.unexpected()
		}
	}
	if p.curr.Type != EndBrace {
		return nil, p.unexpected()
	}
	p.next()
	suffix, err := p.parseWordsInBraces()
	if err != nil {
		return nil, err
	}
	ex.Suffix = suffix
	return ex, nil
}

func (p *Parser) parseRangeBraces(prefix Expander) (Expander, error) {
	parseInt := func() (int, error) {
		if p.curr.Type != Literal {
			return 0, p.unexpected()
		}
		i, err := strconv.Atoi(p.curr.Literal)
		if err == nil {
			p.next()
		}
		return i, err
	}
	ex := ExpandRangeBrace{
		Prefix: prefix,
		Step:   1,
	}
	if p.curr.Type == Literal {
		if n := len(p.curr.Literal); strings.HasPrefix(p.curr.Literal, "0") && n > 1 {
			str := strings.TrimLeft(p.curr.Literal, "0")
			ex.Pad = (n - len(str)) + 1
		}
	}
	var err error
	if ex.From, err = parseInt(); err != nil {
		return nil, err
	}
	if p.curr.Type != Range {
		return nil, p.unexpected()
	}
	p.next()
	if ex.To, err = parseInt(); err != nil {
		return nil, err
	}
	if p.curr.Type == Range {
		p.next()
		if ex.Step, err = parseInt(); err != nil {
			return nil, err
		}
	}
	if p.curr.Type != EndBrace {
		return nil, p.unexpected()
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseSlice(ident Token) (Expander, error) {
	e := ExpandSlice{
		Ident:  ident.Literal,
		Quoted: p.quoted,
	}
	p.next()
	if p.curr.Type == Literal {
		i, err := strconv.Atoi(p.curr.Literal)
		if err != nil {
			return nil, err
		}
		e.Offset = i
		p.next()
	}
	if p.curr.Type != Slice {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type == Literal {
		i, err := strconv.Atoi(p.curr.Literal)
		if err != nil {
			return nil, err
		}
		e.Size = i
		p.next()
	}
	return e, nil
}

func (p *Parser) parsePadding(ident Token) (Expander, error) {
	e := ExpandPad{
		Ident: ident.Literal,
		What:  p.curr.Type,
		With:  " ",
	}
	p.next()
	switch p.curr.Type {
	case Literal:
		e.With = p.curr.Literal
		p.next()
	case Blank:
		e.With = " "
		p.next()
	case Slice:
	default:
		return nil, p.unexpected()
	}
	if p.curr.Type != Slice {
		return nil, p.unexpected()
	}
	p.next()

	size, err := strconv.Atoi(p.curr.Literal)
	if err != nil {
		return nil, err
	}
	e.Len = size
	p.next()
	return e, nil
}

func (p *Parser) parseReplace(ident Token) (Expander, error) {
	e := ExpandReplace{
		Ident:  ident.Literal,
		What:   p.curr.Type,
		Quoted: p.quoted,
	}
	p.next()
	if p.curr.Type != Literal {
		return nil, p.unexpected()
	}
	e.From = p.curr.Literal
	p.next()
	if p.curr.Type != Replace {
		return nil, p.unexpected()
	}
	p.next()
	switch p.curr.Type {
	case Literal:
		e.To = p.curr.Literal
		p.next()
	case EndExp:
	default:
		return nil, p.unexpected()
	}
	return e, nil
}

func (p *Parser) parseTrim(ident Token) (Expander, error) {
	e := ExpandTrim{
		Ident:  ident.Literal,
		What:   p.curr.Type,
		Quoted: p.quoted,
	}
	p.next()
	if p.curr.Type != Literal {
		return nil, p.unexpected()
	}
	e.Trim = p.curr.Literal
	p.next()
	return e, nil
}

func (p *Parser) parseLower(ident Token) (Expander, error) {
	e := ExpandLower{
		Ident:  ident.Literal,
		All:    p.curr.Type == LowerAll,
		Quoted: p.quoted,
	}
	p.next()
	return e, nil
}

func (p *Parser) parseUpper(ident Token) (Expander, error) {
	e := ExpandUpper{
		Ident:  ident.Literal,
		All:    p.curr.Type == UpperAll,
		Quoted: p.quoted,
	}
	p.next()
	return e, nil
}

func (p *Parser) parseExpansion() (Expander, error) {
	p.next()
	if p.curr.Type == Length {
		p.next()
		if p.curr.Type != Literal {
			return nil, p.unexpected()
		}
		ex := ExpandLength{
			Ident: p.curr.Literal,
		}
		p.next()
		if p.curr.Type != EndExp {
			return nil, p.unexpected()
		}
		p.next()
		return ex, nil
	}
	if p.curr.Type != Literal {
		return nil, p.unexpected()
	}
	ident := p.curr
	p.next()
	var (
		ex  Expander
		err error
	)
	switch p.curr.Type {
	case EndExp:
		ex = createVariable(ident.Literal, p.quoted)
	case Slice:
		ex, err = p.parseSlice(ident)
	case TrimSuffix, TrimSuffixLong, TrimPrefix, TrimPrefixLong:
		ex, err = p.parseTrim(ident)
	case Replace, ReplaceAll, ReplacePrefix, ReplaceSuffix:
		ex, err = p.parseReplace(ident)
	case Lower, LowerAll:
		ex, err = p.parseLower(ident)
	case Upper, UpperAll:
		ex, err = p.parseUpper(ident)
	case PadLeft, PadRight:
		ex, err = p.parsePadding(ident)
	case ValIfUnset:
		p.next()
		ex = createValIfUnset(ident.Literal, p.curr.Literal, p.quoted)
		p.next()
	case SetValIfUnset:
		p.next()
		ex = createSetValIfUnset(ident.Literal, p.curr.Literal, p.quoted)
		p.next()
	case ValIfSet:
		p.next()
		ex = createExpandValIfSet(ident.Literal, p.curr.Literal, p.quoted)
		p.next()
	case ExitIfUnset:
		p.next()
		ex = createExpandExitIfUnset(ident.Literal, p.curr.Literal, p.quoted)
		p.next()
	default:
		err = p.unexpected()
	}
	if err != nil {
		return nil, err
	}
	if p.curr.Type != EndExp {
		return nil, p.unexpected()
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseVariable() (ExpandVar, error) {
	ex := createVariable(p.curr.Literal, p.quoted)
	p.next()
	return ex, nil
}

func (p *Parser) enterLoop() {
	p.loop++
}

func (p *Parser) leaveLoop() {
	p.loop--
}

func (p *Parser) inLoop() bool {
	return p.loop > 0
}

func (p *Parser) enterQuote() {
	p.quoted = true
}

func (p *Parser) leaveQuote() {
	p.quoted = false
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) done() bool {
	return p.curr.Type == EOF
}

func (p *Parser) skipBlank() {
	for p.curr.Type == Blank {
		p.next()
	}
}

func (p *Parser) unexpected() error {
	return fmt.Errorf("shell: unexpected token %s", p.curr)
}
