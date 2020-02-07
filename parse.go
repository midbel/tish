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
	typof := kindOr
	if p.curr.Type == tokAnd {
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

	right, err := p.parseCommand()
	if err != nil {
		return nil, err
	}
	is.words = append(is.words, right)
	return is, nil
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
	var xs []Word
	for !p.isDone() {
		if p.curr.Type == tokEOF {
			break
		}
		switch p.curr.Type {
		case tokWord:
			xs = append(xs, Literal(p.curr.Literal))
		case tokVar:
			xs = append(xs, Variable(p.curr.Literal))
		default:
			return nil, fmt.Errorf("unexpected token %s", p.curr)
		}
		p.next()
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
