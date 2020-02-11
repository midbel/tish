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
		plus:          p.parseInfixExpr,
		minus:         p.parseInfixExpr,
		div:           p.parseInfixExpr,
		mul:           p.parseInfixExpr,
		modulo:        p.parseInfixExpr,
		tokLeftShift:  p.parseInfixExpr,
		tokRightShift: p.parseInfixExpr,
	}
	p.prefix = map[rune]func() (Evaluator, error){
		lparen:   p.parsePrefixExpr,
		minus:    p.parsePrefixExpr,
		tokInt:   p.parsePrefixExpr,
		tokFloat: p.parsePrefixExpr,
		tokVar:   p.parsePrefixExpr,
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
			return nil, fmt.Errorf("sequence: unexpected operator: %s", p.curr)
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

func (p *parser) parsePreBraces(prolog Word) (Word, error) {
	p.next()

	ws := List{kind: kindBraces}
	for !p.isDone() && p.curr.Type != tokEndBrace {
		if p.curr.Type != tokWord {
			return nil, fmt.Errorf("braces: %s is not a word", p.curr)
		}
		ws.words = append(ws.words, Literal(p.curr.Literal))

		p.next()
		if p.curr.Type == tokEndBrace {
			break
		}
		switch p.curr.Type {
		case tokBeginBrace:
			n := len(ws.words) - 1
			w, err := p.parsePreBraces(ws.words[n])
			if err != nil {
				return nil, err
			}
			ws.words[n] = w
		case comma:
		default:
			return nil, fmt.Errorf("braces: %s is not allowed", p.curr)
		}
		p.next()
	}
	p.next()

	var w Word
	switch len(ws.words) {
	case 0:
		w = Literal("{}")
	case 1:
		w = ws.words[0]
		if i, ok := w.(Literal); ok {
			w = Literal(fmt.Sprintf("{%s}", string(i)))
		}
	default:
		w = Brace{
			word:   ws,
			prolog: prolog,
		}
	}
	return w, nil
}

func (p *parser) parsePostBraces(w Word) (Word, error) {
	if b, ok := w.(Brace); ok && !p.isBlank() {
		epilog, err := p.parseWord()
		if err != nil {
			return nil, err
		}
		if _, ok := epilog.(Brace); ok {
			w = Brace{
				prolog: b,
				word:   epilog,
			}
		} else {
			b.epilog = epilog
			w = b
		}
	}
	return w, nil
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

func (p *parser) parseInfixExpr(left Evaluator) (Evaluator, error) {
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

func (p *parser) parsePrefixExpr() (Evaluator, error) {
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

func (p *parser) parseAssignment() (Word, error) {
	a := Assignment{ident: p.curr.Literal}
	p.next()
	p.next()

	ws := List{kind: kindSimple}
	for !p.isDone() {
		if p.curr.Type == semicolon {
			break
		}
		w, err := p.parseWord()
		if err != nil {
			return nil, err
		}
		ws.words = append(ws.words, w)
		if p.isBlank() {
			p.next()
		}
	}
	if len(ws.words) > 0 {
		a.word = ws.asWord()
	}
	return a, nil
}

func (p *parser) parseCommand() (Word, error) {
	switch p.curr.Type {
	default:
		return nil, fmt.Errorf("command: unexpected operator %s", p.curr)
	case tokBeginSub:
		return p.parseSubstitution()
	case tokBeginBrace:
		return p.parseWord()
	case tokWord:
	}

	if p.peek.Type == equal {
		return p.parseAssignment()
	}

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
		case tokBeginBrace:
			w, err := p.parsePreBraces(asWord(xs))
			if err != nil {
				return nil, err
			}
			w, err = p.parsePostBraces(w)
			if err != nil {
				return nil, err
			}
			xs = []Word{w}
		default:
			return nil, fmt.Errorf("word: unexpected token %s", p.curr)
		}
		if p.isBlank() || p.isControl() {
			break
		}
	}
	return asWord(xs), nil
}

func asWord(xs []Word) Word {
	var w Word
	if n := len(xs); n == 0 {
	} else if n == 1 {
		w = xs[0]
	} else {
		w = List{words: xs}
	}
	return w
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
