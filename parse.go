package tish

import (
	"fmt"
	"io"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	kws map[string]func() (Command, error)

	loop int
}

func Parse(r io.Reader) (*Parser, error) {
	s, err := NewScanner(r)
	if err != nil {
		return nil, err
	}

	var p Parser
	p.scan = s
	p.kws = map[string]func() (Command, error){
		kwIf:       p.parseIf,
		kwCase:     p.parseCase,
		kwFor:      p.parseFor,
		kwWhile:    p.parseWhile,
		kwUntil:    p.parseUntil,
		kwBreak:    p.parseBC,
		kwContinue: p.parseBC,
	}
	p.next()
	p.next()

	return &p, nil
}

func (p *Parser) Parse() (Command, error) {
	if p.isDone() {
		return nil, io.EOF
	}
	return p.parse()
}

func (p *Parser) parse() (Command, error) {
	switch p.curr.Type {
	case TokKeyword:
		parse, ok := p.kws[p.curr.Literal]
		if !ok {
			return nil, fmt.Errorf("parser: unexpected keyword %s", p.curr)
		}
		return parse()
	case TokLiteral, TokVariable:
		return p.parseSimple()
	default:
		return nil, fmt.Errorf("parser: unexpected token: %s", p.curr)
	}
}

func (p *Parser) parseSimple() (Command, error) {
	var cmd Simple
	for p.peek.Type == TokAssign {
		curr := p.curr
		p.next()
		p.next()
		a, err := p.parseAssign(curr)
		if err != nil {
			return nil, err
		}
		switch p.curr.Type {
		case TokSemicolon, TokEOF:
			p.next()
			return a, nil
		case TokBlank:
			cmd.env = append(cmd.env, a)
		default:
			return nil, fmt.Errorf("parser: unexpected token %s, want 'literal/semicolon'", p.curr)
		}
		p.next()
	}
	for !p.curr.Type.EndOfCommand() {
		w, err := p.parseWord()
		if err != nil {
			return nil, err
		}
		cmd.words = append(cmd.words, w)
		switch p.curr.Type {
		case TokAnd:
			return p.parseAnd(cmd)
		case TokOr:
			return p.parseOr(cmd)
		case TokBlank:
			p.next()
		}
	}
	p.next()
	return cmd, nil
}

func (p *Parser) parseAssign(name Token) (Assign, error) {
	a := Assign{name: name}
	w, err := p.parseWord()
	if err == nil {
		a.word = w
	}
	return a, err
}

func (p *Parser) parseWord() (Word, error) {
	var w Word
	for !p.isDone() && !p.curr.Type.EndOfWord() {
		if !p.curr.Quoted && p.curr.Type == TokKeyword {
			return w, fmt.Errorf("parser: unexpected keyword %s", p.curr)
		}
		w.tokens = append(w.tokens, p.curr)
		p.next()
	}
	return w, nil
}

func (p *Parser) parseAnd(left Command) (Command, error) {
	p.next()
	right, err := p.parseSimple()
	if err == nil {
		a := And{
			left:  left,
			right: right,
		}
		return a, nil
	}
	return nil, err
}

func (p *Parser) parseOr(left Command) (Command, error) {
	p.next()
	right, err := p.parseSimple()
	if err == nil {
		o := Or{
			left:  left,
			right: right,
		}
		return o, nil
	}
	return nil, err
}

func (p *Parser) parseList(stop func(Token) bool) (Command, error) {
	var (
		list   List
		closed bool
	)
	for !p.isDone() {
		c, err := p.parse()
		if err != nil {
			return nil, err
		}
		list.cmds = append(list.cmds, c)
		if closed = stop(p.curr); closed {
			break
		}
	}
	if p.isDone() && !closed {
		return nil, io.ErrUnexpectedEOF
	}
	return list, nil
}

func (p *Parser) parseIf() (Command, error) {
	p.next()
	var (
		cmd If
		err error
	)
	cmd.cmd, err = p.parseList(func(tok Token) bool {
		return tok.Type == TokKeyword && tok.Literal == kwThen
	})
	if err != nil {
		return nil, err
	}
	if p.curr.Type != TokKeyword && p.curr.Literal != kwThen {
		return nil, fmt.Errorf("parser: unexpected token %s, want 'then'", p.curr)
	}
	p.next()

	stop := func(tok Token) bool {
		return tok.Type == TokKeyword && (tok.Literal == kwFi || tok.Literal == kwElse)
	}
	if cmd.csq, err = p.parseList(stop); err != nil {
		return nil, err
	}
	if p.curr.Type != TokKeyword {
		return nil, fmt.Errorf("parser: unexpected token %s, want 'fi/else'", p.curr)
	}
	if p.curr.Literal == kwFi {
		p.next()
		return cmd, nil
	}
	if p.curr.Literal != kwElse {
		return nil, fmt.Errorf("parser: unexpected token %s, want 'else'", p.curr)
	}
	p.next()

	if p.curr.Type == TokKeyword && p.curr.Literal == kwIf {
		cmd.alt, err = p.parseIf()
	} else {
		cmd.alt, err = p.parseList(stop)
		if p.curr.Type != TokKeyword && p.curr.Literal != kwFi {
			return nil, fmt.Errorf("parser: unexpected token %s, want 'fi'", p.curr)
		}
	}
	p.next()

	return cmd, err
}

func (p *Parser) parseCase() (Command, error) {
	p.next()
	return nil, nil
}

func (p *Parser) parseBC() (Command, error) {
	if !p.inLoop() {
		return nil, fmt.Errorf("parser: 'break/continue' not in a loop")
	}
	var cmd Command
	switch p.curr.Literal {
	case kwBreak:
		cmd = Break{}
	case kwContinue:
		cmd = Continue{}
	}
	p.next()
	if p.curr.Type != TokSemicolon {
		return nil, fmt.Errorf("parser: unexpected token %s, want 'newline/semicolon'", p.curr)
	}
	p.next()
	return cmd, nil
}

func (p *Parser) parseFor() (Command, error) {
	p.next()
	var (
		cmd For
		err error
	)
	cmd.name = p.curr
	p.next()
	for p.curr.Type == TokBlank {
		p.next()
	}

	if p.curr.Type != TokKeyword && p.curr.Literal != kwIn {
		return nil, fmt.Errorf("parser: unexpected token %s, want 'in'", p.curr)
	}
	p.next()

	for {
		w, err := p.parseWord()
		if err != nil {
			return nil, err
		}
		cmd.words = append(cmd.words, w)
		if p.curr.Type == TokSemicolon {
			break
		}
		p.next()
	}
	p.next()

	if p.curr.Type != TokKeyword && p.curr.Literal != kwDo {
		return nil, fmt.Errorf("parser: unexpected token %s, want 'do'", p.curr)
	}
	p.next()

	p.enterLoop()
	defer p.leaveLoop()

	cmd.body, err = p.parseList(func(tok Token) bool {
		return tok.Type == TokKeyword && tok.Literal == kwDone
	})
	if err != nil {
		return nil, err
	}
	if p.curr.Type != TokKeyword && p.curr.Literal != kwDone {
		return nil, fmt.Errorf("parser: unexpected token %s, want 'done'", p.curr)
	}
	p.next()

	return cmd, nil
}

func (p *Parser) parseWhile() (Command, error) {
	p.next()
	var (
		cmd While
		err error
	)
	p.enterLoop()
	defer p.leaveLoop()

	cmd.cmd, cmd.body, err = p.parseLoop()
	return cmd, err
}

func (p *Parser) parseUntil() (Command, error) {
	p.next()
	var (
		cmd Until
		err error
	)
	p.enterLoop()
	defer p.leaveLoop()

	cmd.cmd, cmd.body, err = p.parseLoop()
	return cmd, err
}

func (p *Parser) parseLoop() (Command, Command, error) {
	var (
		cmd  Command
		body Command
		err  error
	)
	cmd, err = p.parseList(func(tok Token) bool {
		return tok.Type == TokKeyword && tok.Literal == kwDo
	})
	if err != nil {
		return nil, nil, err
	}
	if p.curr.Type != TokKeyword && p.curr.Literal != kwDo {
		return nil, nil, fmt.Errorf("parser: unexpected token %s, want 'do'", p.curr)
	}
	p.next()

	body, err = p.parseList(func(tok Token) bool {
		return tok.Type == TokKeyword && tok.Literal == kwDone
	})
	if err != nil {
		return nil, nil, err
	}
	if p.curr.Type != TokKeyword && p.curr.Literal != kwDone {
		return nil, nil, fmt.Errorf("parser: unexpected token %s, want 'done'", p.curr)
	}
	p.next()
	return cmd, body, nil
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

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Next()
}

func (p *Parser) isDone() bool {
	return p.curr.Type == TokEOF
}
