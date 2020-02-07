package tish

import (
	"errors"
	"fmt"
	"io"
)

// const (
// 	bindLowest = iota
// 	bindPipe
// 	bindSeq
// 	bindCdt
// )
//
// var bindings = map[rune]int{
// 	tokAnd:    bindCdt,
// 	tokOr:     bindCdt,
// 	pipe:      bindPipe,
// 	semicolon: bindSeq,
// }

type parser struct {
	scan *Scanner
	curr Token
	peek Token

	infix map[rune]func(Word) (Word, error)

	err error
}

func Parse(str string) (Word, error) {
	var p parser
	p.scan = NewScanner(str)
	p.infix = map[rune]func(Word) (Word, error){
		pipe:      p.parsePipeline,
		semicolon: p.parseSequence,
		tokAnd:    p.parseConditional,
		tokOr:     p.parseConditional,
	}
	p.next()
	p.next()

	return p.Parse()
}

func (p *parser) Parse() (Word, error) {
	return p.parse()
}

func (p *parser) parse() (Word, error) {
	left, err := p.parseCommand()
	if err != nil || p.isDone() {
		return left, err
	}
	for !p.isDone() {
		infix, ok := p.infix[p.curr.Type]
		if !ok {
			return nil, fmt.Errorf("unexpected token %s", p.curr)
		}
		left, err = infix(left)
		if err != nil {
			return nil, err
		}
	}
	return left, p.err
}

func (p *parser) parseSequence(left Word) (Word, error) {
	p.next()

	seq := List{
		words: []Word{left},
		kind:  kindSeq,
	}
	right, err := p.parseCommand()
	if err != nil {
		return nil, err
	}
	seq.words = append(seq.words, right)
	return seq, nil
}

func (p *parser) parsePipeline(left Word) (Word, error) {
	p.next()

	pipe := List{
		words: []Word{left},
		kind:  kindPipe,
	}
	right, err := p.parseCommand()
	if err != nil {
		return nil, err
	}
	pipe.words = append(pipe.words, right)
	return pipe, nil
}

func (p *parser) parseConditional(left Word) (Word, error) {
	typof := kindOr
	if p.curr.Type == tokAnd {
		typof = kindAnd
	}
	cdt := List{
		words: []Word{left},
		kind:  typof,
	}

	p.next()

	right, err := p.parseCommand()
	if err != nil {
		return nil, err
	}
	cdt.words = append(cdt.words, right)
	return cdt, nil
}

func (p *parser) parseCommand() (Word, error) {
	ws := List{kind: kindSimple}
	for !p.isControl() {
		w, err := p.parseWord()
		if err != nil {
			return nil, err
		}
		ws.words = append(ws.words, w)
		if p.isBlank() {
			p.next()
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
