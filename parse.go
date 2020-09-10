package tish

import (
	"errors"
	"fmt"
	"io"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	kws map[string]func() (Command, error)

	loop        int
	keepComment bool
	// allow func(Token) bool
}

func Parse(r io.Reader) (Command, error) {
	p, err := NewParser(r)
	if err != nil {
		return nil, err
	}
	var (
		list List
		cmd  Command
	)
	for {
		cmd, err = p.Parse()
		if err != nil {
			break
		}
		list.cmds = append(list.cmds, cmd)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return list, nil
}

func NewParser(r io.Reader) (*Parser, error) {
	s, err := NewScanner(r)
	if err != nil {
		return nil, err
	}

	var p Parser
	p.scan = s
	p.keepComment = false
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
			return nil, fmt.Errorf("parse: unexpected keyword %s", p.curr)
		}
		return parse()
	case TokLiteral, TokVariable:
		return p.parseSimple()
	default:
		return nil, fmt.Errorf("parse: unexpected token: %s", p.curr)
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
			return nil, fmt.Errorf("simple: unexpected token %s, want 'literal/semicolon'", p.curr)
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
	a := Assign{ident: name}
	if p.curr.Type == TokBlank {
		p.next()
		return a, nil
	}
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
			return w, fmt.Errorf("word: unexpected keyword %s", p.curr)
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
	if err := p.parseKeyword(kwThen, true); err != nil {
		return nil, err
	}

	stop := func(tok Token) bool {
		if tok.Type != TokKeyword {
			return false
		}
		return tok.Literal == kwFi || tok.Literal == kwElse || tok.Literal == kwElif
	}
	if cmd.csq, err = p.parseList(stop); err != nil {
		return nil, err
	}
	if err := p.parseKeyword(kwFi, true); err == nil {
		return cmd, nil
	} else if err = p.parseKeyword(kwElif, false); err == nil {
		cmd.alt, err = p.parseIf()
		return cmd, nil
	} else if err = p.parseKeyword(kwElse, true); err == nil {
		cmd.alt, err = p.parseList(func(tok Token) bool {
			return tok.Type == TokKeyword && tok.Literal == kwFi
		})
	} else {
		return nil, fmt.Errorf("if: unexpected token %s, want 'fi/elif/else'", p.curr)
	}
	if err != nil {
		return nil, err
	}
	return cmd, p.parseKeyword(kwFi, true)
}

func (p *Parser) parseCase() (Command, error) {
	p.next()
	var (
		cmd Case
		err error
	)
	cmd.word, err = p.parseWord()
	if err != nil {
		return nil, err
	}
	p.skipBlanks()

	if err := p.parseKeyword(kwIn, true); err != nil {
		return nil, err
	}
	for !p.isDone() {
		c, err := p.parseClause()
		if err != nil {
			return nil, err
		}
		cmd.clauses = append(cmd.clauses, c)
		if p.curr.Type == TokKeyword && p.curr.Literal == kwEsac {
			break
		}
	}
	if p.isDone() {
		return nil, io.ErrUnexpectedEOF
	}
	p.next()
	return cmd, err
}

func (p *Parser) parseClause() (Command, error) {
	var (
		cmd Clause
		err error
	)
	for !p.isDone() {
		if p.curr.Type == TokEndGroup {
			break
		}
		w, err := p.parseWord()
		if err != nil {
			return nil, err
		}
		cmd.pattern = append(cmd.pattern, w)
		switch p.curr.Type {
		case TokPipe:
			p.next()
		case TokEndGroup:
		default:
			return nil, fmt.Errorf("clause: unexpected token %s, want 'pipe/rparen'", p.curr)
		}
	}
	if p.isDone() {
		return nil, io.ErrUnexpectedEOF
	}
	p.next()
	if p.curr.Type == TokSemicolon {
		p.next()
	}
	cmd.body, err = p.parseList(func(tok Token) bool {
		return tok.Type.IsBreak()
	})
	if err != nil {
		return nil, err
	}
	cmd.op = p.curr
	p.next()
	return cmd, nil
}

//069/256211

func (p *Parser) parseBC() (Command, error) {
	if !p.inLoop() {
		return nil, fmt.Errorf("bc: 'break/continue' not in a loop")
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
		return nil, fmt.Errorf("bc: unexpected token %s, want 'newline/semicolon'", p.curr)
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
	cmd.ident = p.curr
	p.next()
	for p.curr.Type == TokBlank {
		p.next()
	}

	if err := p.parseKeyword(kwIn, true); err != nil {
		return nil, err
	}

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
	if err := p.parseKeyword(kwDo, true); err != nil {
		return nil, err
	}

	p.enterLoop()
	defer p.leaveLoop()

	cmd.body, err = p.parseList(func(tok Token) bool {
		return tok.Type == TokKeyword && tok.Literal == kwDone
	})
	if err != nil {
		return nil, err
	}
	if err := p.parseKeyword(kwDone, true); err != nil {
		return nil, err
	}
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
	if err := p.parseKeyword(kwDo, true); err != nil {
		return nil, nil, err
	}

	body, err = p.parseList(func(tok Token) bool {
		return tok.Type == TokKeyword && tok.Literal == kwDone
	})
	if err != nil {
		return nil, nil, err
	}
	return cmd, body, p.parseKeyword(kwDone, true)
}

func (p *Parser) parseKeyword(which string, next bool) error {
	tok := Token{
		Literal: which,
		Type:    TokKeyword,
	}
	var err error
	if !p.curr.Equal(tok) {
		err = fmt.Errorf("keyword: unexpected token %s, want %s", p.curr, tok)
	} else {
		if next {
			p.next()
		}
	}
	return err
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

func (p *Parser) skipBlanks() {
	for p.curr.Type == TokBlank {
		p.next()
	}
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Next()
	if p.keepComment {
		return
	}
	for p.curr.Type == TokComment {
		p.next()
		if p.curr.Type != TokComment {
			p.next()
		}
	}
}

func (p *Parser) isDone() bool {
	return p.curr.Type == TokEOF
}
