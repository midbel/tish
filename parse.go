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

func Parse(r io.Reader) (Word, error) {
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

	p.scan = NewScanner(r)
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
		e = Variable{
			ident:  p.curr.Literal,
			quoted: p.curr.Quoted,
		}
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
		if p.curr.Type != tokPipe && p.isControl() {
			break
		}
		p.next()
		if p.isControl() {
			return nil, fmt.Errorf("command: unexpected operator: %s", p.curr)
		}
	}
	return ws.asWord(), p.err
}

func (p *parser) parseRedirection() (Word, error) {
	var file, mode int
	switch p.curr.Type {
	default:
		return nil, fmt.Errorf("unsupported redirection operator: %s", p.curr)
	case tokRedirectStdin:
		file, mode = fdIn, modRead
	case tokRedirectStdout:
		file, mode = fdOut, modWrite
	case tokRedirectStderr:
		file, mode = fdErr, modWrite
	case tokRedirectBoth:
		file, mode = fdBoth, modWrite
	case tokAppendStdout:
		file, mode = fdOut, modAppend
	case tokAppendStderr:
		file, mode = fdErr, modAppend
	case tokAppendBoth:
		file, mode = fdBoth, modAppend
	case tokRedirectErrToOut:
		file, mode = fdOut, modRelink
	case tokRedirectOutToErr:
		file, mode = fdErr, modRelink
	}
	p.next()

	r := Redirect{
		file: file,
		mode: mode,
	}

	if r.mode == modRelink {
		fmt.Println("parseRedirect: relink", r)
		return r, nil
	}

	var ws []Word
	for {
		if p.isRedirection() || p.isControl() || p.isDone() {
			break
		}
		switch p.curr.Type {
		case tokWord:
			ws = append(ws, Literal(p.curr.Literal))
		case tokVar:
			v := Variable{
				ident:  p.curr.Literal,
				quoted: p.curr.Quoted,
				apply:  Identity(),
			}
			ws = append(ws, v)
		default:
			return nil, fmt.Errorf("redirection: unexpected token type %s", p.curr)
		}
		p.next()
		if p.isBlank() {
			p.next()
		}
	}
	r.Word = asWord(ws)
	fmt.Println("parseRedirect: default", ws)
	return r, nil
}

func (p *parser) parseWord() (Word, error) {
	xs := make([]Word, 0, 10)
	for !p.isDone() {
		if p.curr.Type == tokEOF {
			break
		}
		if p.isRedirection() {
			return p.parseRedirection()
		}
		switch p.curr.Type {
		case tokWord:
			xs = append(xs, Literal(p.curr.Literal))
			p.next()
		case tokVar:
			v := Variable{
				ident:  p.curr.Literal,
				quoted: p.curr.Quoted,
				apply:  Identity(),
			}
			xs = append(xs, v)
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
		case tokBeginParam:
			w, err := p.parseParameter()
			if err != nil {
				return nil, err
			}
			xs = append(xs, w)
		default:
			return nil, fmt.Errorf("word: unexpected token %s", p.curr)
		}
		if p.isBlank() || p.isControl() || p.isRedirection() {
			break
		}
	}
	return asWord(xs), nil
}

func (p *parser) parseParameter() (Word, error) {
	p.next()
	if p.curr.Type == tokVarLength {
		p.next()
		if p.curr.Type != tokVar {
			return nil, fmt.Errorf("parameter: unexpected token: %s", p.curr)
		}
		v := Variable{
			ident:  p.curr.Literal,
			quoted: p.curr.Quoted,
			apply:  Length(),
		}
		p.next()
		if p.curr.Type != tokEndParam {
			return nil, fmt.Errorf("parameter: unexpected token: %s", p.curr)
		}
		p.next()
		return v, nil
	}
	if p.curr.Type != tokVar {
		return nil, fmt.Errorf("parameter: unexpected token: %s", p.curr)
	}
	v := Variable{
		ident:  p.curr.Literal,
		quoted: p.curr.Quoted,
	}
	p.next()
	switch typof := p.curr.Type; typof {
	case tokTrimSuffix, tokTrimSuffixLong:
		p.next()
		if p.curr.Type != tokWord {
			return nil, fmt.Errorf("parameter(suffix): unexpected token %s", p.curr)
		}
		v.apply = TrimSuffix(p.curr.Literal, typof == tokTrimSuffixLong)
		p.next()
	case tokTrimPrefix, tokTrimPrefixLong:
		p.next()
		if p.curr.Type != tokWord {
			return nil, fmt.Errorf("parameter(prefix): unexpected token %s", p.curr)
		}
		v.apply = TrimSuffix(p.curr.Literal, typof == tokTrimPrefixLong)
		p.next()
	case tokReplace, tokReplaceAll, tokReplacePrefix, tokReplaceSuffix:
		var from, to string
		p.next()
		if p.curr.Type != tokWord {
			return nil, fmt.Errorf("parameter(prefix): unexpected token %s", p.curr)
		}
		from = p.curr.Literal
		p.next()
		if p.curr.Type != tokReplace {

		}
		p.next()
		if p.curr.Type != tokWord {
			return nil, fmt.Errorf("parameter(prefix): unexpected token %s", p.curr)
		}
		to = p.curr.Literal
		switch typof {
		case tokReplace:
			v.apply = Replace(from, to)
		case tokReplaceAll:
			v.apply = ReplaceAll(from, to)
		case tokReplacePrefix:
			v.apply = ReplacePrefix(from, to)
		case tokReplaceSuffix:
			v.apply = ReplaceSuffix(from, to)
		}
		p.next()
	case tokGetIfDef, tokGetIfUndef, tokSetIfUndef:
		p.next()
		if p.curr.Type != tokWord {
			return nil, fmt.Errorf("parameter(prefix): unexpected token %s", p.curr)
		}
		switch typof {
		case tokGetIfDef:
			v.apply = GetIfDef(p.curr.Literal)
		case tokGetIfUndef:
			v.apply = GetIfUndef(p.curr.Literal)
		case tokSetIfUndef:
			v.apply = SetIfUndef(p.curr.Literal)
		}
		p.next()
	case tokLower, tokLowerAll:
		p.next()
		v.apply = Lower(typof == tokLowerAll)
	case tokUpper, tokUpperAll:
		v.apply = Lower(typof == tokUpperAll)
	case tokSliceOffset:
		var (
			offset int
			length int
			err    error
		)
		p.next()
		if p.curr.Type != tokInt {
			return nil, fmt.Errorf("parameter(offset): unexpected token %s", p.curr)
		}
		if offset, err = strconv.Atoi(p.curr.Literal); err != nil {
			return nil, err
		}
		p.next()
		if p.curr.Type != tokSliceLen {
			return nil, fmt.Errorf("parameter(length): unexpected token %s", p.curr)
		}
		p.next()
		if length, err = strconv.Atoi(p.curr.Literal); err != nil {
			return nil, err
		}
		p.next()

		v.apply = Substring(offset, length)
	case tokEndParam:
	default:
		return nil, fmt.Errorf("parameter: unexpected token %s", p.curr)
	}
	if p.curr.Type != tokEndParam {
		return nil, fmt.Errorf("parameter: unexpected token: %s", p.curr)
	}
	p.next()
	return v, nil
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

func (p *parser) isRedirection() bool {
	switch p.curr.Type {
	case tokRedirectStdin:
	case tokRedirectStdout:
	case tokRedirectStderr:
	case tokRedirectBoth:
	case tokAppendStdout:
	case tokAppendStderr:
	case tokAppendBoth:
	case tokRedirectErrToOut:
	case tokRedirectOutToErr:
	default:
		return false
	}
	return true
}

func (p *parser) isControl() bool {
	switch p.curr.Type {
	case tokEOF:
	case tokAnd:
	case tokOr:
	case tokPipe:
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
