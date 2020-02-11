package tish

import (
	"fmt"
	"strconv"
	"strings"
)

type Kind int

const (
	kindSimple Kind = iota
	kindSeq
	kindPipe
	kindAnd
	kindOr
	kindList
	kindSub
	kindExpr
	kindBraces
)

func (k Kind) String() string {
	switch k {
	case kindSimple:
		return "simple"
	case kindSeq:
		return "sequence"
	case kindPipe:
		return "pipeline"
	case kindAnd:
		return "and"
	case kindOr:
		return "or"
	case kindList:
		return "list"
	case kindSub:
		return "substitution"
	case kindExpr:
		return "expression"
	default:
		return "unknown"
	}
}

type Word interface {
	Expand(*Env) ([]string, error)
	Equal(Word) bool
	fmt.Stringer

	asWord() Word
}

type List struct {
	words []Word
	kind  Kind
}

func (i List) Expand(e *Env) ([]string, error) {
	ws := make([]string, 0, len(i.words)*4)
	for _, w := range i.words {
		xs, err := w.Expand(e)
		if err != nil {
			return nil, err
		}
		ws = append(ws, xs...)
	}
	return ws, nil
}

func (i List) String() string {
	var buf strings.Builder
	buf.WriteString(i.kind.String())
	buf.WriteRune(lparen)
	for i, w := range i.words {
		if i > 0 {
			buf.WriteRune(comma)
		}
		buf.WriteString(w.String())
	}
	buf.WriteRune(rparen)
	return buf.String()
}

func (i List) Equal(w Word) bool {
	other, ok := w.(List)
	if !ok || i.kind != other.kind || len(i.words) != len(other.words) {
		return false
	}
	for j := 0; j < len(i.words); j++ {
		if !i.words[j].Equal(other.words[j]) {
			return false
		}
	}
	return true
}

func (i List) asWord() Word {
	if i.kind == kindSub || i.kind == kindExpr || len(i.words) != 1 {
		return i
	}
	return i.words[0].asWord()
	// return i.words[0] //.asWord()
}

type Brace struct {
	prolog Word
	epilog Word
	word   Word
}

func (b Brace) Expand(e *Env) ([]string, error) {
	if e.word == nil {
		return nil, nil
	}
	ws, err := b.word.Expand(e)
	if err != nil {
		return nil, err
	}
	for i, w := range []Word{b.prolog, b.epilog} {
		if w == nil {
			continue
		}
		xs, err := w.Expand(e)
		if err != nil {
			return nil, err
		}
		ws = combineWords(ws, xs, i == 0)
	}
	return ws, nil
}

func (b Brace) String() string {
	var buf strings.Builder
	buf.WriteString("brace(")
	if b.prolog != nil {
		buf.WriteString("pre(")
		buf.WriteString(b.prolog.String())
		buf.WriteString(")")
	}

	buf.WriteRune(lcurly)
	buf.WriteString(b.word.String())
	buf.WriteRune(rcurly)

	if b.epilog != nil {
		buf.WriteString("post(")
		buf.WriteString(b.epilog.String())
		buf.WriteString(")")
	}
	buf.WriteString(")")
	return buf.String()
}

func (b Brace) Equal(w Word) bool {
	_, ok := w.(Brace)
	return ok
}

func (b Brace) asWord() Word {
	return b
}

type Variable string

func (v Variable) Expand(e *Env) ([]string, error) {
	return e.Get(string(v))
}

func (v Variable) String() string {
	return fmt.Sprintf("variable(%s)", string(v))
}

func (v Variable) Equal(w Word) bool {
	other, ok := w.(Variable)
	if !ok {
		return false
	}
	return string(other) == string(v)
}

func (v Variable) Eval(e *Env) (Number, error) {
	vs, err := e.Get(string(v))
	if err != nil {
		return 0, err
	}
	x, err := strconv.ParseInt(vs[0], 10, 64)
	if err != nil {
		return 0, err
	}
	return Number(x), nil
}

func (v Variable) asWord() Word {
	return v
}

type Literal string

func (i Literal) Expand(_ *Env) ([]string, error) {
	return []string{string(i)}, nil
}

func (i Literal) String() string {
	return fmt.Sprintf("literal(%s)", string(i))
}

func (i Literal) Equal(w Word) bool {
	other, ok := w.(Literal)
	if !ok {
		return false
	}
	return string(other) == string(i)
}

func (i Literal) asWord() Word {
	return i
}

type Number int64

func (n Number) Eval(_ *Env) (Number, error) {
	return n, nil
}

func (n Number) Expand(_ *Env) ([]string, error) {
	x := strconv.FormatInt(int64(n), 10)
	return []string{x}, nil
}

func (n Number) String() string {
	return fmt.Sprintf("number(%d)", int64(n))
}

func (n Number) Equal(w Word) bool {
	other, ok := w.(Number)
	if !ok {
		return ok
	}
	return int64(n) == int64(other)
}

func (n Number) asWord() Word {
	return n
}

func combineWords(ws, ps []string, prefix bool) []string {
	var words []string
	for i := range ws {
		for j := range ps {
			if prefix {
				words = append(words, ps[j]+ws[i])
			} else {
				words = append(words, ws[i]+ps[j])
			}
		}
	}
	if len(words) == 0 {
		words = ws
	}
	return words
}
