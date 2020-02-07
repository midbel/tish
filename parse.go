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
	p := parser{
		scan: NewScanner(str),
	}
	p.next()
	p.next()

	return p.Parse()
}

func (p *parser) Parse() (Word, error) {
	var ws List
	for !p.isDone() {
		w, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		ws.words = append(ws.words, w)
		p.next()
	}
	return ws.asWord(), p.err
}

func (p *parser) parseCommand() (Word, error) {
	var ws List
	for {
		w, err := p.parseWord()
		if err != nil {
			return nil, err
		}
		if !p.isControl() {
			break
		}
		ws.words = append(ws.words, w)
		p.next()
	}
	return ws, nil
}

func (p *parser) parseWord() (Word, error) {
	var xs []Word
	for !(p.isBlank() || p.isControl()) {
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
	}
	var w Word
	if n := len(xs); n == 0 {
		w = Literal("")
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
