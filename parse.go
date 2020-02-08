package tish

import (
	"errors"
	"fmt"
	"io"
)

type parser struct {
	scan *Scanner
	curr Token
	peek Token

	err error
}

func Parse(str string) (Word, error) {
	var p parser

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

func (p *parser) parseExpression() (Word, error) {
	p.next()
	ws := List{kind: kindExpr}
	for {
		if p.curr.Type == tokEndArith {
			break
		}
		p.next()
	}
	p.next()
	return ws, nil
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
			w, err := p.parseExpression()
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
