package tish

import (
	"fmt"
	"io"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	kws    map[string]func() (Command, error)
	prefix map[Kind]func() (Evaluator, error)
	infix  map[Kind]func(Evaluator) (Evaluator, error)

	loop        int
	keepComment bool
}

// func Parse(r io.Reader) (Command, error) {
// 	p, err := NewParser(r)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var (
// 		list List
// 		cmd  Command
// 	)
// 	for {
// 		cmd, err = p.Parse()
// 		if err != nil {
// 			break
// 		}
// 		list.cmds = append(list.cmds, cmd)
// 	}
// 	if err != nil && !errors.Is(err, io.EOF) {
// 		return nil, err
// 	}
// 	return list, nil
// }

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
	p.prefix = map[Kind]func() (Evaluator, error){
		TokIncr:     p.parsePrefix,
		TokDecr:     p.parsePrefix,
		TokLiteral:  p.parsePrefix,
		TokVariable: p.parsePrefix,
		TokNumber:   p.parsePrefix,
		TokSub:      p.parsePrefix,
		TokNot:      p.parsePrefix,
		TokBinNot:   p.parsePrefix,
		TokBegGroup: p.parsePrefix,
	}
	p.infix = map[Kind]func(Evaluator) (Evaluator, error){
		TokAdd:              p.parseInfix,
		TokSub:              p.parseInfix,
		TokMul:              p.parseInfix,
		TokDiv:              p.parseInfix,
		TokMod:              p.parseInfix,
		TokExponent:         p.parseInfix,
		TokLeftShift:        p.parseInfix,
		TokRightShift:       p.parseInfix,
		TokBinAnd:           p.parseInfix,
		TokBinOr:            p.parseInfix,
		TokBinXor:           p.parseInfix,
		TokEqual:            p.parseInfix,
		TokNotEqual:         p.parseInfix,
		TokLesser:           p.parseInfix,
		TokLessEq:           p.parseInfix,
		TokGreater:          p.parseInfix,
		TokGreatEq:          p.parseInfix,
		TokAnd:              p.parseInfix,
		TokOr:               p.parseInfix,
		TokAssign:           p.parseInfix,
		TokAddAssign:        p.parseInfix,
		TokSubAssign:        p.parseInfix,
		TokMulAssign:        p.parseInfix,
		TokDivAssign:        p.parseInfix,
		TokModAssign:        p.parseInfix,
		TokLeftShiftAssign:  p.parseInfix,
		TokRightShiftAssign: p.parseInfix,
		TokBinAndAssign:     p.parseInfix,
		TokBinOrAssign:      p.parseInfix,
		TokBinXorAssign:     p.parseInfix,
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
	p.skipBlanks()

	switch {
	default:
		return nil, fmt.Errorf("parse: unexpected token: %s", p.curr)
	case p.curr.isKeyword():
		parse, ok := p.kws[p.curr.Literal]
		if !ok {
			return nil, fmt.Errorf("parse: unexpected keyword %s", p.curr)
		}
		return parse()
	case p.curr.isSimple():
		return p.parseSimple()
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

func (p *Parser) parseLength() Word {
	p.next()
	return Length{ident: p.curr}
}

func (p *Parser) parseVariable() Word {
	return Literal{token: p.curr}
}

func (p *Parser) parseTransform() Word {
	w := Transform{
		ident: p.curr,
		op:    p.peek,
	}
	p.next()
	return w
}

func (p *Parser) parseTrim() Word {
	t := Trim{
		ident: p.curr,
		part:  p.peek,
	}
	p.next()
	p.next()
	t.str = p.curr
	return t
}

func (p *Parser) parseReplace() (Word, error) {
	r := Replace{ident: p.curr, op: p.peek}
	p.next()
	p.next()
	if p.curr.Type != TokVariable && p.curr.Type != TokLiteral && p.curr.Type != TokNumber {
		return nil, fmt.Errorf("replace: unexpected token %s, want 'number/variable/literal'", p.curr)
	}
	r.src = p.curr
	p.next()
	if p.curr.Type != TokVariable && p.curr.Type != TokLiteral && p.curr.Type != TokNumber {
		return nil, fmt.Errorf("replace: unexpected token %s, want 'number/variable/literal'", p.curr)
	}
	r.dst = p.curr
	return r, nil
}

func (p *Parser) parseSlice() (Word, error) {
	i := Slice{ident: p.curr}
	p.next()
	p.next()
	if p.curr.Type != TokNumber && p.curr.Type != TokVariable {
		return nil, fmt.Errorf("slice: unexpected token %s, want 'number/variable'", p.curr)
	}
	i.offset = p.curr
	p.next()
	if p.curr.Type != TokNumber && p.curr.Type != TokVariable {
		return nil, fmt.Errorf("slice: unexpected token %s, want 'number/variable'", p.curr)
	}
	i.length = p.curr
	return i, nil
}

func (p *Parser) parseExpansion() (Word, error) {
	p.next()
	var (
		w   Word
		err error
	)
	switch k := p.curr.Type; {
	case k == TokLen:
		w = p.parseLength()
	case k == TokVariable && p.peek.Type == TokEndExp:
		w = p.parseVariable()
	case k == TokVariable && p.peek.isTrim():
		w = p.parseTrim()
	case k == TokVariable && p.peek.isReplace():
		w, err = p.parseReplace()
	case k == TokVariable && p.peek.isTransform():
		w = p.parseTransform()
	case k == TokVariable && p.peek.isSlice():
		w, err = p.parseSlice()
	default:
		return w, fmt.Errorf("expansion: unexpected token %s", p.curr)
	}
	if err != nil {
		return nil, err
	}
	p.next()
	if p.curr.Type != TokEndExp {
		return w, fmt.Errorf("expansion: unexpected token %s", p.curr)
	}
	p.next()
	return w, nil
}

func (p *Parser) parseSerie(s Serie) (Word, error) {
	for !p.isDone() {
		w, err := p.parseWord()
		if err != nil {
			return nil, err
		}
		s.words = append(s.words, w)
		if p.curr.Type == TokEndBrace {
			p.next()
			break
		}
		if p.curr.Type != TokSerie {
			return nil, fmt.Errorf("serie: unexpected token %s, want 'comma'", p.curr)
		}
		p.next()
	}
	return s, nil
}

func (p *Parser) parseRange(r Range) (Word, error) {
	if p.curr.Type != TokNumber && p.curr.Type != TokVariable {
		return nil, fmt.Errorf("range(first): unexpected token %s, want 'number|variable'", p.curr)
	}
	r.first = p.curr
	p.next()
	if p.curr.Type != TokRange {
		return nil, fmt.Errorf("range: unexpected token %s, want 'range'", p.curr)
	}
	p.next()
	if p.curr.Type != TokNumber && p.curr.Type != TokVariable {
		return nil, fmt.Errorf("range(last): unexpected token %s, want 'number|variable'", p.curr)
	}
	r.last = p.curr
	p.next()
	if p.curr.Type == TokEndBrace {
		p.next()
		r.incr = Token{Literal: "1", Type: TokNumber}
		return r, nil
	}
	if p.curr.Type != TokRange {
		return nil, fmt.Errorf("range: unexpected token %s, want 'range'", p.curr)
	}
	p.next()
	if p.curr.Type != TokNumber && p.curr.Type != TokVariable {
		return nil, fmt.Errorf("range(incr): unexpected token %s, want 'number|variable'", p.curr)
	}
	r.incr = p.curr

	p.next()
	if p.curr.Type != TokEndBrace {
		return nil, fmt.Errorf("range: unexpected token %s, want 'brace'", p.curr)
	}
	p.next()

	return r, nil
}

func (p *Parser) parseBraces(prefix Word) (Word, error) {
	p.next()
	switch p.peek.Type {
	case TokSerie:
		s := Serie{prefix: prefix}
		return p.parseSerie(s)
	case TokRange:
		r := Range{prefix: prefix}
		return p.parseRange(r)
	default:
		return nil, fmt.Errorf("brace: unexpected token %s, want 'serie|range'", p.peek)
	}
}

func (p *Parser) parseArithmetic() (Word, error) {
	p.next()
	var es EvalList
	for p.curr.Type != TokEndArith {
		e, err := p.parseExpression(bindLowest)
		if err != nil {
			return nil, err
		}
		es.evals = append(es.evals, e)
		if p.curr.Type == TokSemicolon {
			p.next()
		}
	}
	if p.curr.Type != TokEndArith {
		return nil, fmt.Errorf("expr: unexpected token %s, want %s", p.curr, TokEndArith)
	}
	p.next()
	return Expr{eval: es.asEvaluator()}, nil
}

func (p *Parser) parseExpression(bp int) (Evaluator, error) {
	prefix, ok := p.prefix[p.curr.Type]
	if !ok {
		return nil, fmt.Errorf("expr(prefix): unexpected operator %s", p.curr)
	}
	left, err := prefix()
	if err != nil {
		return nil, err
	}
	for (p.curr.Type != TokEndArith && p.curr.Type != TokSemicolon) && bp < bindPower(p.curr.Type) {
		infix, ok := p.infix[p.curr.Type]
		if !ok {
			return nil, fmt.Errorf("expr(infix): unexpected operator %s", p.curr)
		}
		left, err = infix(left)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) parsePrefix() (Evaluator, error) {
	var (
		eval Evaluator
		err  error
	)
	switch p.curr.Type {
	case TokLiteral, TokVariable:
		eval = Identifier{ident: p.curr}
		p.next()
	case TokNumber:
		eval = Number{ident: p.curr}
		p.next()
	case TokSub, TokIncr, TokDecr:
		op := p.curr.Type
		p.next()
		eval, err = p.parseExpression(bindPrefix)
		if err == nil {
			eval = Prefix{
				op:    op,
				right: eval,
			}
		}
	case TokBegGroup:
		p.next()
		eval, err = p.parseExpression(bindLowest)
		if err == nil {
			if p.curr.Type != TokEndGroup {
				return nil, fmt.Errorf("expr(prefix): unexpected token %s, want %s", p.curr.Type, TokEndGroup)
			}
			p.next()
		}
	}
	return eval, err
}

func (p *Parser) parseInfix(left Evaluator) (Evaluator, error) {
	if p.curr.Type == TokAssign {
		if _, ok := left.(Identifier); !ok {
			return nil, fmt.Errorf("expr(infix): expecter identifier on left side of assignment (%s)", left)
		}
	}
	i := Infix{
		left: left,
		op:   p.curr.Type,
	}
	p.next()
	right, err := p.parseExpression(bindPower(i.op))
	if err != nil {
		return nil, err
	}
	i.right = right
	return i, nil
}

func (p *Parser) parseAssign(name Token) (Assign, error) {
	a := Assign{ident: name}
	if p.curr.Type == TokBlank || p.curr.Type == TokEOF || p.curr.Type == TokSemicolon {
		a.word = Literal{}
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
	var ws WordList
	for !p.isDone() {
		var (
			w Word
			err error
		)
		switch p.curr.Type {
		case TokLiteral, TokVariable:
			w = Literal{token: p.curr}
		case TokBegArith:
			w, err = p.parseArithmetic()
		case TokBegBrace:
			w, err = p.parseBraces(ws.asWord())
		case TokBegExp:
			w, err = p.parseExpansion()
		case TokKeyword:
			return nil, fmt.Errorf("word: unexpected keyword %s", p.curr)
		default:
			return ws.asWord(), nil
		}
		if err != nil {
			return nil, err
		}
		ws.words = append(ws.words, w)
		p.next()
	}
	return ws.asWord(), nil
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

func (p *Parser) parseClause() (Clause, error) {
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
			return cmd, err
		}
		cmd.pattern = append(cmd.pattern, w)
		switch p.curr.Type {
		case TokPipe:
			p.next()
		case TokEndGroup:
		default:
			return cmd, fmt.Errorf("clause: unexpected token %s, want 'pipe/rparen'", p.curr)
		}
	}
	if p.isDone() {
		return cmd, io.ErrUnexpectedEOF
	}
	p.next()
	if p.curr.Type == TokSemicolon {
		p.next()
	}
	cmd.body, err = p.parseList(func(tok Token) bool {
		return tok.Type.IsBreak()
	})
	if err != nil {
		return cmd, err
	}
	cmd.op = p.curr
	p.next()
	return cmd, nil
}

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
	p.peek = p.scan.Scan()
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
