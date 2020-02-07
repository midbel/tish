package tish

import (
	"fmt"
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
		return "subsitution"
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
	if len(i.words) == 1 {
		return i.words[0]
	}
	return i
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
