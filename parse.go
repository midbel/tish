package tish

import (
	"fmt"
	"io"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token
}

func Parse(r io.Reader) (*Parser, error) {
	s, err := NewScanner(r)
	if err != nil {
		return nil, err
	}

	var p Parser
	p.scan = s
	p.next()
	p.next()

	return &p, nil
}

func (p *Parser) Parse() (Command, error) {
	if p.isDone() {
		return nil, io.EOF
	}
	switch p.curr.Type {
	case TokKeyword:
		return nil, nil
	case TokLiteral:
		return p.parseCommand()
	default:
		return nil, fmt.Errorf("unexpected token: %s", p.curr)
	}
}

func (p *Parser) parseCommand() (Command, error) {
	var s Simple
	for !p.isDone() && p.curr.Type != TokSemicolon {
		var w Word
		for !p.isDone() && p.curr.Type != TokBlank {
			w.tokens = append(w.tokens, p.curr)
			p.next()
		}
		s.words = append(s.words, w)
		p.next()
	}
	p.next()
	return s, nil
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Next()
}

func (p *Parser) isDone() bool {
	return p.curr.Type == TokEOF
}
