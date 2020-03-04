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
	return p.parseSequence(tokEOF)
}

func (p *parser) parseSequence(delimiter rune) (Word, error) {
	ws := List{kind: kindSeq}

	for p.curr.Type != delimiter && !p.isDone() {
		if p.isComment() {
			p.next()
			p.next()
			continue
		}
		if p.curr.Type == tokBeginList {
			w, err := p.parseSubshell()
			if err != nil {
				return nil, err
			}
			ws.words = append(ws.words, w)
			continue
		}
		w, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		switch p.curr.Type {
		case delimiter:
		case tokOr, tokAnd:
			w, err = p.parseConditional(w)
		case semicolon:
		default:
			return nil, fmt.Errorf("sequence(%s): unexpected operator: %s", p.curr.Position, p.curr)
		}
		if err != nil {
			return nil, err
		}
		ws.words = append(ws.words, w)
		if p.curr.Type != delimiter {
			p.next()
		}
	}
	return ws.asWord(), nil
}

func (p *parser) parseSubshell() (Word, error) {
	p.next()

	w, err := p.parseSequence(tokEndList)
	if err != nil {
		return nil, err
	}
	if p.curr.Type != tokEndList {
		return nil, fmt.Errorf("subshell(%s): unexpected token %s", p.curr.Position, p.curr)
	}
	p.next()

	w = List{
		kind:  kindShell,
		words: []Word{w},
	}
	if p.curr.Type == semicolon {
		p.next()
	}
	return w, nil
}

func (p *parser) parseCommand() (Word, error) {
	switch p.curr.Type {
	default:
		return nil, fmt.Errorf("command(%s): unexpected operator %s", p.curr.Position, p.curr)
	case tokBeginSub:
		return p.parseSubstitution()
	case tokBeginBrace:
		return p.parseWord()
	case tokWord:
	}

	ws := List{kind: kindSimple}
	for p.peek.Type == equal {
		w, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}
		ws.words = append(ws.words, w)
	}

	if p.isDone() || p.curr.Type == semicolon {
		return ws.asWord(), nil
	}

	for !p.isDone() {
		w, err := p.parseSimple()
		if err != nil {
			return nil, err
		}
		if p.isPipe() {
			var kind Kind
			switch p.curr.Type {
			case tokPipe:
				kind = kindPipe
			case tokPipeBoth:
				kind = kindPipeBoth
			default:
				return nil, fmt.Errorf("command(%s): unexpected token %s", p.curr.Position, p.curr)
			}
			w = Pipe{
				Word: w,
				kind: kind,
			}
			ws.kind = kindPipe
		}
		ws.words = append(ws.words, w)
		if !p.isPipe() && p.isControl() {
			break
		}

		p.next()
		if p.isControl() {
			return nil, fmt.Errorf("command(%s): unexpected operator: %s", p.curr.Position, p.curr)
		}
	}
	return ws.asWord(), p.err
}

func (p *parser) parseSimple() (Word, error) {
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
		if p.isComment() {
			p.next()
		}
	}
	return xs.asWord(), nil
}

func (p *parser) parseWord() (Word, error) {
	xs := make([]Word, 0, 10)
	for !p.isDone() {
		if p.isRedirection() {
			return p.parseRedirection()
		}
		switch p.curr.Type {
		case tokWord:
			xs = append(xs, Literal(p.curr.Literal))
			p.next()
		case equal:
			xs = append(xs, Literal("="))
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
		case tokComment:
		default:
			return nil, fmt.Errorf("word(%s): unexpected token %s", p.curr.Position, p.curr)
		}
		if p.isBlank() || p.isControl() || p.isRedirection() || p.isComment() {
			break
		}
	}
	return asWord(xs), nil
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
		return nil, fmt.Errorf("condition(%s): unexpected operator: %s", p.curr.Position, p.curr)
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
		if !p.isWord() {
			return nil, fmt.Errorf("braces(%s): %s not a word", p.curr.Position, p.curr)
		}
		ws.words = append(ws.words, Literal(p.curr.Literal))
		p.next()

		if p.curr.Type == tokBeginBrace {
			n := len(ws.words) - 1
			w, err := p.parsePreBraces(ws.words[n])
			if err != nil {
				return nil, err
			}
			ws.words[n] = w
		}

		if p.curr.Type == comma {
			p.next()
			continue
		}
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
	if b, ok := w.(Brace); ok && p.isWord() {
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
	quoted := p.curr.Quoted
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
			return nil, fmt.Errorf("substitution(%s): unexpected operator: %s", p.curr.Position, p.curr)
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
	if quoted {
		w.kind |= kindQuoted
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
		return nil, fmt.Errorf("expr(%s): unexpected prefix operator: %s", p.curr.Position, p.curr)
	}
	left, err := prefix()
	if err != nil {
		return nil, err
	}
	for p.curr.Type != tokEndArith && bp < bindPower(p.curr) {
		infix, ok := p.infix[p.curr.Type]
		if !ok {
			return nil, fmt.Errorf("expr(%s): unexpected infix operator: %s", p.curr.Position, p.curr)
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
				return nil, fmt.Errorf("prefix(%s): unexpected token %s", p.peek.Position, p.peek)
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

	w, err := p.parseWord()
	if err == nil {
		a.word = w
		if p.isBlank() {
			p.next()
		}
	}
	return a, err
}

func (p *parser) parseRedirection() (Word, error) {
	var file, mode int
	switch p.curr.Type {
	default:
		return nil, fmt.Errorf("redirection(%s): unsupported redirection operator %s", p.curr.Position, p.curr)
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
		return r, nil
	}

	switch p.curr.Type {
	case tokWord:
		r.Word = Literal(p.curr.Literal)
	case tokVar:
		r.Word = Variable{
			ident:  p.curr.Literal,
			quoted: p.curr.Quoted,
			apply:  Identity(),
		}
	default:
		return nil, fmt.Errorf("redirection(%s): unexpected token type %s", p.curr.Position, p.curr)
	}
	p.next()
	if p.isBlank() {
		p.next()
	}
	return r, nil
}

func (p *parser) parseParameter() (Word, error) {
	p.next()
	if p.curr.Type == tokVarLength {
		p.next()
		if p.curr.Type != tokVar {
			return nil, fmt.Errorf("parameter(%s): unexpected token: %s", p.curr.Position, p.curr)
		}
		v := Variable{
			ident:  p.curr.Literal,
			quoted: p.curr.Quoted,
			apply:  Length(),
		}
		p.next()
		if p.curr.Type != tokEndParam {
			return nil, fmt.Errorf("parameter(length:%s): unexpected token: %s", p.curr.Position, p.curr)
		}
		p.next()
		return v, nil
	}
	if p.curr.Type != tokVar {
		return nil, fmt.Errorf("parameter(%s): unexpected token: %s", p.curr.Position, p.curr)
	}
	v := Variable{
		ident:  p.curr.Literal,
		quoted: p.curr.Quoted,
	}

	nextWord := func() (Word, error) {
		var (
			w   Word
			err error
		)
		switch p.curr.Type {
		case tokInt:
			x, err := strconv.ParseInt(p.curr.Literal, 0, 64)
			if err != nil {
				return nil, err
			}
			w = Number(x)
		case tokWord:
			w = Literal(p.curr.Literal)
		case tokVar:
			w = Variable{
				ident:  p.curr.Literal,
				quoted: p.curr.Quoted,
				apply:  Identity(),
			}
		case tokBeginSub:
			w, err = p.parseSubstitution()
		case tokBeginArith:
			w, err = p.parseArithmetic()
		default:
			err = fmt.Errorf("parameter(%s): unexpected token %s", p.curr.Position, p.curr)
		}
		return w, err
	}

	p.next()
	switch typof := p.curr.Type; typof {
	case tokTrimSuffix, tokTrimSuffixLong:
		p.next()
		w, err := nextWord()
		if err != nil {
			return nil, fmt.Errorf("parameter(suffix): %s", err)
		}
		v.apply = TrimSuffix(w, typof == tokTrimSuffixLong)
		p.next()
	case tokTrimPrefix, tokTrimPrefixLong:
		p.next()
		w, err := nextWord()
		if err != nil {
			return nil, fmt.Errorf("parameter(prefix): %s", err)
		}
		v.apply = TrimPrefix(w, typof == tokTrimPrefixLong)
		p.next()
	case tokReplace, tokReplaceAll, tokReplacePrefix, tokReplaceSuffix:
		var (
			from Word
			to   Word
			err  error
		)
		p.next()
		if from, err = nextWord(); err != nil {
			return nil, fmt.Errorf("parameter(replace): %s", err)
		}
		p.next()
		if p.curr.Type != tokReplace {
			return nil, fmt.Errorf("parameter(replace): %s", err)
		}
		p.next()
		if to, err = nextWord(); err != nil {
			return nil, err
		}
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
		w, err := nextWord()
		if err != nil {
			return nil, fmt.Errorf("parameter(def/undef): %s", err)
		}
		switch typof {
		case tokGetIfDef:
			v.apply = GetIfDef(w)
		case tokGetIfUndef:
			v.apply = GetIfUndef(w)
		case tokSetIfUndef:
			v.apply = SetIfUndef(w)
		}
		p.next()
	case tokLower, tokLowerAll:
		p.next()
		v.apply = Lower(typof == tokLowerAll)
	case tokUpper, tokUpperAll:
		v.apply = Lower(typof == tokUpperAll)
	case tokSliceOffset:
		var (
			offset Word
			length Word
			err    error
		)
		p.next()
		if offset, err = nextWord(); err != nil {
			return nil, fmt.Errorf("parameter(offset): %s", err)
		}
		p.next()
		if p.curr.Type != tokSliceLen {
			return nil, fmt.Errorf("parameter(length:%s): unexpected token %s", p.curr.Position, p.curr)
		}
		p.next()
		if length, err = nextWord(); err != nil {
			return nil, fmt.Errorf("parameter(length): %s", err)
		}
		p.next()

		v.apply = Substring(offset, length)
	case tokEndParam:
	default:
		return nil, fmt.Errorf("parameter(%s): unexpected token %s", p.curr.Position, p.curr)
	}
	if p.curr.Type != tokEndParam {
		return nil, fmt.Errorf("parameter(%s): unexpected token: %s", p.curr.Position, p.curr)
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
		w = List{words: xs, kind: kindWord}
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

func (p *parser) isPipe() bool {
	return p.curr.Type == tokPipe || p.curr.Type == tokPipeBoth
}

func (p *parser) isComment() bool {
	return p.curr.Type == tokComment
}

func (p *parser) isControl() bool {
	switch p.curr.Type {
	case tokEOF:
	case tokAnd:
	case tokOr:
	case tokPipe:
	case tokPipeBoth:
	case tokEndSub:
	case tokEndList:
	case tokEndArith:
	default:
		return isControl(p.curr.Type)
	}
	return true
}

func (p *parser) isEOL() bool {
	return p.curr.Type == semicolon
}

func (p *parser) isDone() bool {
	return p.err != nil || p.curr.Equal(eof)
}

func (p *parser) isWord() bool {
	return p.curr.Type == tokWord || p.curr.Type == tokInt || p.curr.Type == tokFloat
}

func (p *parser) next() {
	p.curr = p.peek
	peek, err := p.scan.Scan()
	if err != nil && !errors.Is(err, io.EOF) {
		p.err = err
	}
	p.peek = peek
}
