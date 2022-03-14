package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/midbel/tish/internal/token"
	"github.com/midbel/tish/internal/words"
)

type Parser struct {
	scan *Scanner
	curr token.Token
	peek token.Token

	quoted bool
	prefix map[rune]func() (words.Expr, error)
	infix  map[rune]func(words.Expr) (words.Expr, error)

	unary  map[rune]func() (words.Expander, error)
	binary map[rune]func(words.Expander) (words.Expander, error)

	loop int
}

func Parse(str string) (words.Executer, error) {
	p := NewParser(strings.NewReader(str))
	return p.Parse()
}

func NewParser(r io.Reader) *Parser {
	var p Parser
	p.scan = Scan(r)

	p.prefix = map[rune]func() (words.Expr, error){
		token.BegMath:  p.parseUnary,
		token.Numeric:  p.parseUnary,
		token.Variable: p.parseUnary,
		token.Sub:      p.parseUnary,
		token.Inc:      p.parseUnary,
		token.Dec:      p.parseUnary,
		token.BitNot:   p.parseUnary,
		token.Not:      p.parseUnary,
	}
	p.infix = map[rune]func(words.Expr) (words.Expr, error){
		token.Add:        p.parseBinary,
		token.Sub:        p.parseBinary,
		token.Mul:        p.parseBinary,
		token.Div:        p.parseBinary,
		token.Pow:        p.parseBinary,
		token.LeftShift:  p.parseBinary,
		token.RightShift: p.parseBinary,
		token.BitAnd:     p.parseBinary,
		token.BitOr:      p.parseBinary,
		token.BitXor:     p.parseBinary,
		token.BitNot:     p.parseBinary,
		token.Eq:         p.parseBinary,
		token.Ne:         p.parseBinary,
		token.Lt:         p.parseBinary,
		token.Le:         p.parseBinary,
		token.Gt:         p.parseBinary,
		token.Ge:         p.parseBinary,
		token.And:        p.parseBinary,
		token.Or:         p.parseBinary,
		token.Cond:       p.parseTernary,
		token.Assign:     p.parseAssign,
	}

	p.binary = map[rune]func(words.Expander) (words.Expander, error){
		token.Eq:        p.parseBinaryTest,
		token.Ne:        p.parseBinaryTest,
		token.Lt:        p.parseBinaryTest,
		token.Le:        p.parseBinaryTest,
		token.Gt:        p.parseBinaryTest,
		token.Ge:        p.parseBinaryTest,
		token.And:       p.parseBinaryTest,
		token.Or:        p.parseBinaryTest,
		token.NewerThan: p.parseBinaryTest,
		token.OlderThan: p.parseBinaryTest,
		token.SameFile:  p.parseBinaryTest,
	}
	p.unary = map[rune]func() (words.Expander, error){
		token.Not:         p.parseUnaryTest,
		token.BegMath:     p.parseUnaryTest,
		token.FileExists:  p.parseUnaryTest,
		token.FileRead:    p.parseUnaryTest,
		token.FileLink:    p.parseUnaryTest,
		token.FileDir:     p.parseUnaryTest,
		token.FileWrite:   p.parseUnaryTest,
		token.FileSize:    p.parseUnaryTest,
		token.FileRegular: p.parseUnaryTest,
		token.FileExec:    p.parseUnaryTest,
		token.StrNotEmpty: p.parseUnaryTest,
		token.StrEmpty:    p.parseUnaryTest,
		token.Literal:     p.parseUnaryTest,
		token.Variable:    p.parseUnaryTest,
		token.Quote:       p.parseUnaryTest,
	}

	p.next()
	p.next()

	return &p
}

func (p *Parser) Parse() (words.Executer, error) {
	if p.done() {
		return nil, io.EOF
	}
	ex, err := p.parse()
	if err != nil {
		return nil, err
	}
	switch p.curr.Type {
	case token.List, token.Comment, token.EOF:
		p.next()
	default:
		return nil, p.unexpected()
	}
	return ex, nil
}

func (p *Parser) parse() (words.Executer, error) {
	switch {
	case p.peek.Type == token.Assign:
		return p.parseAssignment()
	case p.curr.Type == token.Keyword:
		return p.parseKeyword()
	case p.curr.Type == token.BegTest:
		return p.parseTest()
	case p.curr.Type == token.BegSub:
		return p.parseSubshell()
	default:
	}
	ex, err := p.parseSimple()
	if err != nil {
		return nil, err
	}
	for {
		switch p.curr.Type {
		case token.And:
			return p.parseAnd(ex)
		case token.Or:
			return p.parseOr(ex)
		case token.Pipe, token.PipeBoth:
			ex, err = p.parsePipe(ex)
			if err != nil {
				return nil, err
			}
		default:
			return ex, nil
		}
	}
}

func (p *Parser) parseSubshell() (words.Executer, error) {
	p.next()
	var list words.ExecSubshell
	for !p.done() && p.curr.Type != token.EndSub {
		if p.curr.Type == token.List {
			p.next()
		}
		p.skipBlank()
		x, err := p.parse()
		if err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	if p.curr.Type != token.EndSub {
		return nil, p.unexpected()
	}
	p.next()
	return list.Executer(), nil
}

func (p *Parser) parseTest() (words.Executer, error) {
	p.next()
	ex, err := p.parseTester(words.BindLowest)
	if err != nil {
		return nil, err
	}
	if p.curr.Type != token.EndTest {
		return nil, p.unexpected()
	}
	p.next()

	var test words.ExecTest
	if x, ok := ex.(words.Tester); ok {
		test.Tester = x
	} else {
		test.Tester = words.SingleTest{
			Expander: ex,
		}
	}
	return test, err
}

func (p *Parser) parseTester(pow words.Bind) (words.Expander, error) {
	parse, ok := p.unary[p.curr.Type]
	if !ok {
		return nil, p.unexpected()
	}
	left, err := parse()
	if err != nil {
		return nil, err
	}
	for (p.curr.Type != token.EndMath && p.curr.Type != token.EndTest) && pow < words.BindPower(p.curr) {
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

func (p *Parser) parseUnaryTest() (words.Expander, error) {
	var (
		ex  words.Expander
		err error
	)
	switch op := p.curr.Type; op {
	case token.Variable:
		ex, err = p.parseVariable()
	case token.Literal:
		ex, err = p.parseLiteral()
	case token.Quote:
		ex, err = p.parseQuote()
		p.skipBlank()
	case token.BegMath:
		p.next()
		ex, err = p.parseTester(words.BindLowest)
		if err != nil {
			return nil, err
		}
		if p.curr.Type != token.EndMath {
			err = p.unexpected()
			break
		}
		p.next()
	case token.Not, token.FileExists, token.FileRead, token.FileWrite, token.FileExec, token.FileSize, token.FileLink, token.FileDir, token.FileRegular, token.StrEmpty, token.StrNotEmpty:
		p.next()
		ex, err = p.parseTester(words.BindPrefix)
		if err != nil {
			break
		}
		ex = words.UnaryTest{
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

func (p *Parser) parseBinaryTest(left words.Expander) (words.Expander, error) {
	b := words.BinaryTest{
		Left: left,
		Op:   p.curr.Type,
	}
	w := words.BindPower(p.curr)
	p.next()

	right, err := p.parseTester(w)
	if err == nil {
		b.Right = right
	}
	return b, err
}

func (p *Parser) parseSimple() (words.Executer, error) {
	var (
		ex   words.ExpandList
		dirs []words.ExpandRedirect
	)
	for {
		switch p.curr.Type {
		case token.Literal, token.Quote, token.Variable, token.BegExp, token.BegBrace, token.BegSub, token.BegMath:
			next, err := p.parseWords()
			if err != nil {
				return nil, err
			}
			ex.List = append(ex.List, next)
		case token.RedirectIn, token.RedirectOut, token.RedirectErr, token.RedirectBoth, token.AppendOut, token.AppendBoth:
			next, err := p.parseRedirection()
			if err != nil {
				return nil, err
			}
			dirs = append(dirs, next)
		default:
			sg := words.CreateSimple(ex)
			sg.Redirect = append(sg.Redirect, dirs...)
			return sg, nil
		}
	}
}

func (p *Parser) parseRedirection() (words.ExpandRedirect, error) {
	kind := p.curr.Type
	p.next()
	e, err := p.parseWords()
	if err != nil {
		return words.ExpandRedirect{}, err
	}
	return words.CreateRedirect(e, kind), nil
}

func (p *Parser) parseAssignment() (words.Executer, error) {
	if p.curr.Type != token.Literal {
		return nil, p.unexpected()
	}
	var (
		ident = p.curr.Literal
		list  words.ExpandList
	)
	p.next()
	if p.curr.Type != token.Assign {
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
	return words.CreateAssign(ident, list), nil
}

func (p *Parser) parsePipe(left words.Executer) (words.Executer, error) {
	var list []words.PipeItem
	list = append(list, words.CreatePipeItem(left, p.curr.Type == token.PipeBoth))
	for !p.done() {
		if p.curr.Type != token.Pipe && p.curr.Type != token.PipeBoth {
			break
		}
		var (
			both = p.curr.Type == token.PipeBoth
			ex   words.Executer
			err  error
		)
		p.next()
		if ex, err = p.parseSimple(); err != nil {
			return nil, err
		}
		list = append(list, words.CreatePipeItem(ex, both))
	}
	return words.CreatePipe(list), nil
}

func (p *Parser) parseAnd(left words.Executer) (words.Executer, error) {
	p.next()
	right, err := p.parse()
	if err != nil {
		return nil, err
	}
	return words.CreateAnd(left, right), nil
}

func (p *Parser) parseOr(left words.Executer) (words.Executer, error) {
	p.next()
	right, err := p.parse()
	if err != nil {
		return nil, err
	}
	return words.CreateOr(left, right), nil
}

func (p *Parser) parseKeyword() (words.Executer, error) {
	var (
		ex  words.Executer
		err error
	)
	switch p.curr.Literal {
	case token.KwBreak:
		if !p.inLoop() {
			return nil, p.unexpected()
		}
		ex = words.ExecBreak{}
		p.next()
	case token.KwContinue:
		if !p.inLoop() {
			return nil, p.unexpected()
		}
		ex = words.ExecContinue{}
		p.next()
	case token.KwFor:
		ex, err = p.parseFor()
	case token.KwWhile:
		ex, err = p.parseWhile()
	case token.KwUntil:
		ex, err = p.parseUntil()
	case token.KwIf:
		ex, err = p.parseIf()
	case token.KwCase:
		ex, err = p.parseCase()
	default:
		err = p.unexpected()
	}
	return ex, err
}

func (p *Parser) parseWhile() (words.Executer, error) {
	p.enterLoop()
	defer p.leaveLoop()

	p.next()
	p.skipBlank()
	var (
		ex  words.ExecWhile
		err error
	)
	if ex.Cond, err = p.parse(); err != nil {
		return nil, err
	}
	if p.curr.Type != token.List {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type != token.Keyword || p.curr.Literal != token.KwDo {
		return nil, p.unexpected()
	}
	ex.Body, err = p.parseBody(func(kw string) bool { return kw == token.KwElse || kw == token.KwDone })
	if err != nil {
		return nil, err
	}
	if p.curr.Type == token.Keyword && p.curr.Literal == token.KwElse {
		ex.Alt, err = p.parseBody(func(kw string) bool { return kw == token.KwDone })
		if err != nil {
			return nil, err
		}
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseUntil() (words.Executer, error) {
	p.enterLoop()
	defer p.leaveLoop()

	p.next()
	p.skipBlank()
	var (
		ex  words.ExecUntil
		err error
	)
	if ex.Cond, err = p.parse(); err != nil {
		return nil, err
	}
	if p.curr.Type != token.List {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type != token.Keyword || p.curr.Literal != token.KwDo {
		return nil, p.unexpected()
	}
	ex.Body, err = p.parseBody(func(kw string) bool { return kw == token.KwElse || kw == token.KwDone })
	if err != nil {
		return nil, err
	}
	if p.curr.Type == token.Keyword && p.curr.Literal == token.KwElse {
		ex.Alt, err = p.parseBody(func(kw string) bool { return kw == token.KwDone })
		if err != nil {
			return nil, err
		}
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseIf() (words.Executer, error) {
	p.next()
	p.skipBlank()

	var (
		ex  words.ExecIf
		err error
	)
	if ex.Cond, err = p.parse(); err != nil {
		return nil, err
	}
	if p.curr.Type != token.List {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type != token.Keyword || p.curr.Literal != token.KwThen {
		return nil, p.unexpected()
	}
	ex.Csq, err = p.parseBody(func(kw string) bool { return kw == token.KwElse || kw == token.KwFi })
	if err != nil {
		return nil, err
	}
	if p.curr.Type == token.Keyword && p.curr.Literal == token.KwElse {
		p.next()
		ex.Alt, err = p.parse()
		if err != nil {
			return nil, err
		}
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseClause() (words.ExecClause, error) {
	var c words.ExecClause
	if p.curr.Literal == "*" && p.peek.Type != token.EndSub {
		return c, p.unexpected()
	}
	for !p.done() && p.curr.Type != token.EndSub {
		var word words.Expander
		switch p.curr.Type {
		case token.Literal:
			word, _ = p.parseLiteral()
		case token.Variable:
			word, _ = p.parseVariable()
		default:
			return c, p.unexpected()
		}
		c.List = append(c.List, word)
		switch p.curr.Type {
		case token.Comma:
			p.next()
		case token.EndSub:
		default:
			return c, p.unexpected()
		}
	}
	if p.curr.Type != token.EndSub {
		return c, p.unexpected()
	}
	p.next()
	var list words.ExecList
	for !p.done() {
		if p.curr.Type == token.List {
			p.next()
			p.skipBlank()
		}
		if p.peek.Type == token.Comma || p.peek.Type == token.EndSub || (p.curr.Type == token.Keyword && p.curr.Literal == token.KwEsac) {
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

func (p *Parser) parseCase() (words.Executer, error) {
	p.next()
	var ex words.ExecCase
	switch p.curr.Type {
	case token.Literal:
		ex.Word, _ = p.parseLiteral()
	case token.Variable:
		ex.Word, _ = p.parseVariable()
	default:
		return nil, p.unexpected()
	}
	p.next()
	p.skipBlank()
	if p.curr.Type != token.Keyword && p.curr.Literal != token.KwIn {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type != token.List {
		return nil, p.unexpected()
	}
	p.next()
	for p.curr.Type != token.Keyword && p.curr.Literal != token.KwEsac {
		fallback := p.curr.Type == token.Literal && p.curr.Literal == "*"
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
	if p.curr.Type != token.Keyword && p.curr.Literal != token.KwEsac {
		return nil, p.unexpected()
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseFor() (words.Executer, error) {
	p.enterLoop()
	defer p.leaveLoop()

	p.next()
	p.skipBlank()
	if p.curr.Type != token.Literal {
		return nil, p.unexpected()
	}
	ex := words.ExecFor{
		Ident: p.curr.Literal,
	}
	p.next()
	p.skipBlank()
	if p.curr.Type != token.Keyword || p.curr.Literal != token.KwIn {
		return nil, p.unexpected()
	}
	p.next()
	p.skipBlank()
	for !p.done() && p.curr.Type != token.List {
		e, err := p.parseWords()
		if err != nil {
			return nil, err
		}
		ex.List = append(ex.List, e)
	}
	if p.curr.Type != token.List {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type != token.Keyword && p.curr.Literal != token.KwDo {
		return nil, p.unexpected()
	}
	var err error
	ex.Body, err = p.parseBody(func(kw string) bool { return kw == token.KwElse || kw == token.KwDone })
	if err != nil {
		return nil, err
	}
	if p.curr.Type == token.Keyword && p.curr.Literal == token.KwElse {
		ex.Alt, err = p.parseBody(func(kw string) bool { return kw == token.KwDone })
		if err != nil {
			return nil, err
		}
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseBody(stop func(kw string) bool) (words.Executer, error) {
	var list words.ExecList
	p.next()
	for !p.done() && !stop(p.curr.Literal) {
		switch p.curr.Type {
		case token.Blank:
			p.skipBlank()
			continue
		case token.List:
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
		case token.List, token.Comment:
			p.next()
		case token.Keyword:
			if !stop(p.curr.Literal) {
				return nil, p.unexpected()
			}
		default:
			return nil, p.unexpected()
		}
	}
	if p.curr.Type != token.Keyword || !stop(p.curr.Literal) {
		return nil, p.unexpected()
	}
	return list.Executer(), nil
}

func (p *Parser) parseWords() (words.Expander, error) {
	var list words.ExpandMulti
	for !p.done() {
		if p.curr.Eow() {
			if !p.curr.IsSequence() {
				p.next()
			}
			break
		}
		var (
			next words.Expander
			err  error
		)
		switch p.curr.Type {
		case token.Literal:
			next, err = p.parseLiteral()
		case token.Variable:
			next, err = p.parseVariable()
		case token.Quote:
			next, err = p.parseQuote()
		case token.BegExp:
			next, err = p.parseExpansion()
		case token.BegSub:
			next, err = p.parseSubstitution()
		case token.BegMath:
			next, err = p.parseArithmetic()
		case token.BegBrace:
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

func (p *Parser) parseArithmetic() (words.Expander, error) {
	p.next()
	var list words.ExpandMath
	list.Quoted = p.quoted
	for !p.done() && p.curr.Type != token.EndMath {
		next, err := p.parseExpression(words.BindLowest)
		if err != nil {
			return nil, err
		}
		switch p.curr.Type {
		case token.List:
			p.next()
		case token.EndMath:
		default:
			return nil, p.unexpected()
		}
		list.List = append(list.List, next)
	}
	if p.curr.Type != token.EndMath {
		return nil, p.unexpected()
	}
	p.next()
	return list, nil
}

func (p *Parser) parseExpression(pow words.Bind) (words.Expr, error) {
	fn, ok := p.prefix[p.curr.Type]
	if !ok {
		return nil, p.unexpected()
	}
	left, err := fn()
	if err != nil {
		return nil, err
	}
	for (p.curr.Type != token.EndMath && p.curr.Type != token.List) && pow < words.BindPower(p.curr) {
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

func (p *Parser) parseUnary() (words.Expr, error) {
	var (
		ex  words.Expr
		err error
	)
	switch p.curr.Type {
	case token.Sub, token.Inc, token.Dec, token.Not, token.BitNot:
		op := p.curr.Type
		p.next()
		ex, err = p.parseExpression(words.BindPrefix)
		if err != nil {
			break
		}
		ex = words.CreateUnary(ex, op)
	case token.BegMath:
		p.next()
		ex, err = p.parseExpression(words.BindLowest)
		if err != nil {
			break
		}
		if p.curr.Type != token.EndMath {
			err = p.unexpected()
			break
		}
		p.next()
	case token.Numeric:
		ex = words.CreateNumber(p.curr.Literal)
		p.next()
	case token.Variable:
		ex = words.CreateVariable(p.curr.Literal, false)
		p.next()
	default:
		return nil, p.unexpected()
	}
	return ex, err
}

func (p *Parser) parseBinary(left words.Expr) (words.Expr, error) {
	b := words.Binary{
		Left: left,
		Op:   p.curr.Type,
	}
	w := words.BindPower(p.curr)
	p.next()

	right, err := p.parseExpression(w)
	if err == nil {
		b.Right = right
	}
	return b, err
}

func (p *Parser) parseAssign(left words.Expr) (words.Expr, error) {
	var as words.Assignment
	switch v := left.(type) {
	case words.ExpandVar:
		as.Ident = v.Ident
	default:
		return nil, p.unexpected()
	}
	p.next()

	expr, err := p.parseExpression(words.BindLowest)
	if err != nil {
		return nil, err
	}
	as.Expr = expr
	return as, nil
}

func (p *Parser) parseTernary(left words.Expr) (words.Expr, error) {
	p.next()
	ter := words.Ternary{
		Cond: left,
	}
	left, err := p.parseExpression(words.BindTernary)
	if err != nil {
		return nil, err
	}
	ter.Left = left
	if p.curr.Type != token.Alt {
		return nil, p.unexpected()
	}
	p.next()
	right, err := p.parseExpression(words.BindLowest)
	if err != nil {
		return nil, err
	}
	ter.Right = right
	return ter, nil
}

func (p *Parser) parseSubstitution() (words.Expander, error) {
	var ex words.ExpandSub
	ex.Quoted = p.quoted
	p.next()
	for !p.done() && p.curr.Type != token.EndSub {
		next, err := p.parse()
		if err != nil {
			return nil, err
		}
		ex.List = append(ex.List, next)
	}
	if p.curr.Type != token.EndSub {
		return nil, p.unexpected()
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseQuote() (words.Expander, error) {
	p.enterQuote()
	p.next()

	var list words.ExpandMulti
	for !p.done() && p.curr.Type != token.Quote {
		var (
			next words.Expander
			err  error
		)
		switch p.curr.Type {
		case token.Literal:
			next, err = p.parseLiteral()
		case token.Variable:
			next, err = p.parseVariable()
		case token.BegExp:
			next, err = p.parseExpansion()
		case token.BegSub:
			next, err = p.parseSubstitution()
		case token.BegMath:
			next, err = p.parseArithmetic()
		default:
			err = p.unexpected()
		}
		if err != nil {
			return nil, err
		}
		list.List = append(list.List, next)
	}
	if p.curr.Type != token.Quote {
		return nil, p.unexpected()
	}
	p.leaveQuote()
	p.next()
	return list.Expander(), nil
}

func (p *Parser) parseLiteral() (words.ExpandWord, error) {
	ex := words.CreateWord(p.curr.Literal, p.quoted)
	p.next()
	return ex, nil
}

func (p *Parser) parseBraces(prefix words.Expander) (words.Expander, error) {
	p.next()
	if p.peek.Type == token.Range {
		return p.parseRangeBraces(prefix)
	}
	return p.parseListBraces(prefix)
}

func (p *Parser) parseWordsInBraces() (words.Expander, error) {
	var list words.ExpandList
	for !p.done() {
		if p.curr.Type == token.Seq || p.curr.Type == token.EndBrace {
			break
		}
		var (
			next words.Expander
			err  error
		)
		switch p.curr.Type {
		case token.Literal:
			next, err = p.parseLiteral()
		case token.BegBrace:
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

func (p *Parser) parseListBraces(prefix words.Expander) (words.Expander, error) {
	ex := words.ExpandListBrace{
		Prefix: prefix,
	}
	for !p.done() {
		if p.curr.Type == token.EndBrace {
			break
		}
		x, err := p.parseWordsInBraces()
		if err != nil {
			return nil, err
		}
		ex.Words = append(ex.Words, x)
		switch p.curr.Type {
		case token.Seq:
			p.next()
		case token.EndBrace:
		default:
			return nil, p.unexpected()
		}
	}
	if p.curr.Type != token.EndBrace {
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

func (p *Parser) parseRangeBraces(prefix words.Expander) (words.Expander, error) {
	parseInt := func() (int, error) {
		if p.curr.Type != token.Literal {
			return 0, p.unexpected()
		}
		i, err := strconv.Atoi(p.curr.Literal)
		if err == nil {
			p.next()
		}
		return i, err
	}
	ex := words.ExpandRangeBrace{
		Prefix: prefix,
		Step:   1,
	}
	if p.curr.Type == token.Literal {
		if n := len(p.curr.Literal); strings.HasPrefix(p.curr.Literal, "0") && n > 1 {
			str := strings.TrimLeft(p.curr.Literal, "0")
			ex.Pad = (n - len(str)) + 1
		}
	}
	var err error
	if ex.From, err = parseInt(); err != nil {
		return nil, err
	}
	if p.curr.Type != token.Range {
		return nil, p.unexpected()
	}
	p.next()
	if ex.To, err = parseInt(); err != nil {
		return nil, err
	}
	if p.curr.Type == token.Range {
		p.next()
		if ex.Step, err = parseInt(); err != nil {
			return nil, err
		}
	}
	if p.curr.Type != token.EndBrace {
		return nil, p.unexpected()
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseSlice(ident token.Token) (words.Expander, error) {
	e := words.ExpandSlice{
		Ident:  ident.Literal,
		Quoted: p.quoted,
	}
	p.next()
	if p.curr.Type == token.Literal {
		i, err := strconv.Atoi(p.curr.Literal)
		if err != nil {
			return nil, err
		}
		e.Offset = i
		p.next()
	}
	if p.curr.Type != token.Slice {
		return nil, p.unexpected()
	}
	p.next()
	if p.curr.Type == token.Literal {
		i, err := strconv.Atoi(p.curr.Literal)
		if err != nil {
			return nil, err
		}
		e.Size = i
		p.next()
	}
	return e, nil
}

func (p *Parser) parsePadding(ident token.Token) (words.Expander, error) {
	e := words.ExpandPad{
		Ident: ident.Literal,
		What:  p.curr.Type,
		With:  " ",
	}
	p.next()
	switch p.curr.Type {
	case token.Literal:
		e.With = p.curr.Literal
		p.next()
	case token.Blank:
		e.With = " "
		p.next()
	case token.Slice:
	default:
		return nil, p.unexpected()
	}
	if p.curr.Type != token.Slice {
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

func (p *Parser) parseReplace(ident token.Token) (words.Expander, error) {
	e := words.ExpandReplace{
		Ident:  ident.Literal,
		What:   p.curr.Type,
		Quoted: p.quoted,
	}
	p.next()
	if p.curr.Type != token.Literal {
		return nil, p.unexpected()
	}
	e.From = p.curr.Literal
	p.next()
	if p.curr.Type != token.Replace {
		return nil, p.unexpected()
	}
	p.next()
	switch p.curr.Type {
	case token.Literal:
		e.To = p.curr.Literal
		p.next()
	case token.EndExp:
	default:
		return nil, p.unexpected()
	}
	return e, nil
}

func (p *Parser) parseTrim(ident token.Token) (words.Expander, error) {
	e := words.ExpandTrim{
		Ident:  ident.Literal,
		What:   p.curr.Type,
		Quoted: p.quoted,
	}
	p.next()
	if p.curr.Type != token.Literal {
		return nil, p.unexpected()
	}
	e.Trim = p.curr.Literal
	p.next()
	return e, nil
}

func (p *Parser) parseLower(ident token.Token) (words.Expander, error) {
	e := words.ExpandLower{
		Ident:  ident.Literal,
		All:    p.curr.Type == token.LowerAll,
		Quoted: p.quoted,
	}
	p.next()
	return e, nil
}

func (p *Parser) parseUpper(ident token.Token) (words.Expander, error) {
	e := words.ExpandUpper{
		Ident:  ident.Literal,
		All:    p.curr.Type == token.UpperAll,
		Quoted: p.quoted,
	}
	p.next()
	return e, nil
}

func (p *Parser) parseExpansion() (words.Expander, error) {
	p.next()
	if p.curr.Type == token.Length {
		p.next()
		if p.curr.Type != token.Literal {
			return nil, p.unexpected()
		}
		ex := words.ExpandLength{
			Ident: p.curr.Literal,
		}
		p.next()
		if p.curr.Type != token.EndExp {
			return nil, p.unexpected()
		}
		p.next()
		return ex, nil
	}
	if p.curr.Type != token.Literal {
		return nil, p.unexpected()
	}
	ident := p.curr
	p.next()
	var (
		ex  words.Expander
		err error
	)
	switch p.curr.Type {
	case token.EndExp:
		ex = words.CreateVariable(ident.Literal, p.quoted)
	case token.Slice:
		ex, err = p.parseSlice(ident)
	case token.TrimSuffix, token.TrimSuffixLong, token.TrimPrefix, token.TrimPrefixLong:
		ex, err = p.parseTrim(ident)
	case token.Replace, token.ReplaceAll, token.ReplacePrefix, token.ReplaceSuffix:
		ex, err = p.parseReplace(ident)
	case token.Lower, token.LowerAll:
		ex, err = p.parseLower(ident)
	case token.Upper, token.UpperAll:
		ex, err = p.parseUpper(ident)
	case token.PadLeft, token.PadRight:
		ex, err = p.parsePadding(ident)
	case token.ValIfUnset:
		p.next()
		ex = words.CreateValIfUnset(ident.Literal, p.curr.Literal, p.quoted)
		p.next()
	case token.SetValIfUnset:
		p.next()
		ex = words.CreateSetValIfUnset(ident.Literal, p.curr.Literal, p.quoted)
		p.next()
	case token.ValIfSet:
		p.next()
		ex = words.CreateExpandValIfSet(ident.Literal, p.curr.Literal, p.quoted)
		p.next()
	case token.ExitIfUnset:
		p.next()
		ex = words.CreateExpandExitIfUnset(ident.Literal, p.curr.Literal, p.quoted)
		p.next()
	default:
		err = p.unexpected()
	}
	if err != nil {
		return nil, err
	}
	if p.curr.Type != token.EndExp {
		return nil, p.unexpected()
	}
	p.next()
	return ex, nil
}

func (p *Parser) parseVariable() (words.ExpandVar, error) {
	ex := words.CreateVariable(p.curr.Literal, p.quoted)
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
	return p.curr.Type == token.EOF
}

func (p *Parser) skipBlank() {
	for p.curr.Type == token.Blank {
		p.next()
	}
}

func (p *Parser) unexpected() error {
	return fmt.Errorf("shell: unexpected token %s", p.curr)
}
