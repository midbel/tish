package tish

import (
	"errors"
	"fmt"
	"io"
	"strconv"
)

type parser struct {
	scan *Scanner
	curr Token
	peek Token

	err error

	infix  map[rune]func(Evaluator) (Evaluator, error)
	prefix map[rune]func() (Evaluator, error)
}

func Parse(str string) (Word, error) {
	var p parser

	p.infix = map[rune]func(Evaluator) (Evaluator, error){
		plus:   p.parseInfix,
		minus:  p.parseInfix,
		div:    p.parseInfix,
		mul:    p.parseInfix,
		modulo: p.parseInfix,
	}
	p.prefix = map[rune]func() (Evaluator, error){
		lparen:   p.parsePrefix,
		minus:    p.parsePrefix,
		tokInt:   p.parsePrefix,
		tokFloat: p.parsePrefix,
		tokVar:   p.parsePrefix,
	}

	p.scan = NewScanner(str)
	p.next()
	p.next()

	return p.Parse()
}

func (p *parser) Parse() (Word, error) {
	return p.parseSequence()
}

func (p *parser) parseSequence() (Word, error) {
	ws := List{kind: kindSeq}

	for !p.isDone() {
		w, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		switch p.curr.Type {
		case tokEOF:
		case tokOr, tokAnd:
			w, err = p.parseConditional(w)
		case semicolon:
		default:
			return nil, fmt.Errorf("unexpected operator: %s", p.curr)
		}
		if err != nil {
			return nil, err
		}
		ws.words = append(ws.words, w)
		p.next()
	}
	return ws.asWord(), nil
}

func (p *parser) parseConditional(left Word) (Word, error) {
	typof, token := kindOr, p.curr.Type
	if token == tokAnd {
		typof = kindAnd
	}
	is := List{
		words: []Word{left},
		kind:  typof,
	}

	p.next()
	if p.isControl() {
		return nil, fmt.Errorf("cdt: unexpected operator: %s", p.curr)
	}

	for {
		right, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		is.words = append(is.words, right)

		tok := p.curr
		if p.isControl() && !(tok.Type == tokAnd || tok.Type == tokOr) {
			break
		}
		if tok.Type != token {
			return p.parseConditional(is)
		}
		p.next()
	}
	return is, nil
}

func (p *parser) parseSubstitution() (Word, error) {
	p.next()

	ws := List{kind: kindSeq}
	for {
		w, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		switch p.curr.Type {
		case tokEndSub:
		case tokOr, tokAnd:
			w, err = p.parseConditional(w)
		case semicolon:
		default:
			return nil, fmt.Errorf("substitution: unexpected operator: %s", p.curr)
		}
		if err != nil {
			return nil, err
		}
		ws.words = append(ws.words, w)
		if p.curr.Type == tokEndSub {
			break
		}
		p.next()
	}
	p.next()
	w := List{
		words: []Word{ws.asWord()},
		kind:  kindSub,
	}
	return w, nil
}

func (p *parser) parseArithmetic() (Word, error) {
	p.next()

	e, err := p.parseExpression(bindLowest)
	if err != nil {
		return nil, err
	}
	ws := List{
		kind:  kindExpr,
		words: []Word{Expr{expr: e}},
	}
	p.next()
	return ws, nil
}

func (p *parser) parseExpression(bp int) (Evaluator, error) {
	prefix, ok := p.prefix[p.curr.Type]
	if !ok {
		return nil, fmt.Errorf("expr: unexpected prefix operator: %s", p.curr)
	}
	left, err := prefix()
	if err != nil {
		return nil, err
	}
	for p.curr.Type != tokEndArith && bp < bindPower(p.curr) {
		infix, ok := p.infix[p.curr.Type]
		if !ok {
			return nil, fmt.Errorf("expr: unexpected infix operator: %s", p.curr)
		}
		left, err = infix(left)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *parser) parseInfix(left Evaluator) (Evaluator, error) {
	e := infix{
		left: left,
		op:   p.curr.Type,
	}

	bp := bindPower(p.curr)
	p.next()

	right, err := p.parseExpression(bp)
	if err == nil {
		e.right = right
	}
	return e, err
}

func (p *parser) parsePrefix() (Evaluator, error) {
	var (
		e   Evaluator
		err error
	)
	switch p.curr.Type {
	case lparen:
		p.next()
		e, err = p.parseExpression(bindLowest)
		if err == nil {
			if p.curr.Type != rparen {
				return nil, fmt.Errorf("unexpected token: %s", p.peek)
			}
			p.next()
		}
	case minus:
		p.next()
		e, err = p.parseExpression(bindPrefix)
		if err == nil {
			e = prefix{right: e, op: minus}
		}
	case tokVar:
		e = Variable(p.curr.Literal)
		p.next()
	default:
		n, err := strconv.ParseInt(p.curr.Literal, 10, 64)
		if err == nil {
			e = Number(n)
			p.next()
		}
	}
	return e, err
}

func (p *parser) parseCommand() (Word, error) {
	ws := List{kind: kindPipe}
	for {
		xs := List{kind: kindSimple}
		for !p.isControl() {
			w, err := p.parseWord()
			if err != nil {
				return nil, err
			}
			xs.words = append(xs.words, w)
			if p.isBlank() {
				p.next()
			}
		}
		ws.words = append(ws.words, xs.asWord())
		if p.curr.Type != pipe && p.isControl() {
			break
		}
		p.next()
		if p.isControl() {
			return nil, fmt.Errorf("command: unexpected operator: %s", p.curr)
		}
	}
	return ws.asWord(), p.err
}

func (p *parser) parseWord() (Word, error) {
	xs := make([]Word, 0, 10)
	for !p.isDone() {
		if p.curr.Type == tokEOF {
			break
		}
		switch p.curr.Type {
		case tokWord:
			xs = append(xs, Literal(p.curr.Literal))
			p.next()
		case tokVar:
			xs = append(xs, Variable(p.curr.Literal))
			p.next()
		case tokBeginSub:
			w, err := p.parseSubstitution()
			if err != nil {
				return nil, err
			}
			xs = append(xs, w)
		case tokBeginArith:
			w, err := p.parseArithmetic()
			if err != nil {
				return nil, err
			}
			xs = append(xs, w)
		default:
			return nil, fmt.Errorf("word: unexpected token %s", p.curr)
		}
		if p.isBlank() || p.isControl() {
			break
		}
	}

	var w Word
	if n := len(xs); n == 0 {
	} else if n == 1 {
		w = xs[0]
	} else {
		w = List{words: xs}
	}
	return w, nil
}

func (p *parser) isBlank() bool {
	return p.curr.Type == tokBlank || p.curr.Type == tokEOF
}

func (p *parser) isControl() bool {
	switch p.curr.Type {
	case tokEOF:
	case tokAnd:
	case tokOr:
	case tokEndSub:
	case tokEndArith:
	default:
		return isControl(p.curr.Type)
	}
	return true
}

func (p *parser) next() {
	p.curr = p.peek
	peek, err := p.scan.Scan()
	if err != nil && !errors.Is(err, io.EOF) {
		p.err = err
	}
	p.peek = peek
}

func (p *parser) isDone() bool {
	return p.err != nil || p.curr.Equal(eof)
}
