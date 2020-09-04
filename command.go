package tish

import (
	"fmt"
	"strings"
)

type Command interface {
	Execute() (int, error)
	Equal(Command) bool
}

type Word struct {
	tokens []Token
}

func (w Word) String() string {
	ws := make([]string, len(w.tokens))
	for i := range w.tokens {
		ws[i] = w.tokens[i].Literal
	}
	return strings.Join(ws, "")
}

func (w Word) Equal(other Word) bool {
	if len(w.tokens) != len(other.tokens) {
		return false
	}
	for i, t := range w.tokens {
		if !t.Equal(other.tokens[i]) {
			return false
		}
	}
	return true
}

type Simple struct {
	words []Word
}

func (s Simple) Execute() (int, error) {
	return 0, nil
}

func (s Simple) Equal(other Command) bool {
	i, ok := other.(Simple)
	if !ok {
		return false
	}
	if len(s.words) != len(i.words) {
		return false
	}
	for j, w := range s.words {
		if !w.Equal(i.words[j]) {
			return false
		}
	}
	return true
}

func (s Simple) String() string {
	ws := make([]string, len(s.words))
	for i := range s.words {
		ws[i] = s.words[i].String()
	}
	return fmt.Sprintf("simple(%s)", strings.Join(ws, " "))
}

type And struct {
	left  Command
	right Command
}

func (a And) Execute() (int, error) {
	e, err := a.left.Execute()
	if e == 0 && err == nil {
		e, err = a.right.Execute()
	}
	return e, err
}

func (a And) Equal(other Command) bool {
	i, ok := other.(And)
	if !ok {
		return ok
	}
	return a.left.Equal(i.left) && a.right.Equal(i.right)
}

func (a And) String() string {
	return fmt.Sprintf("and(%s, %s)", a.left, a.right)
}

type Or struct {
	left  Command
	right Command
}

func (o Or) Execute() (int, error) {
	e, err := o.left.Execute()
	if e == 0 && err == nil {
		return e, err
	}
	return o.right.Execute()
}

func (o Or) Equal(other Command) bool {
	i, ok := other.(Or)
	if !ok {
		return ok
	}
	return o.left.Equal(i.left) && o.right.Equal(i.right)
}

func (o Or) String() string {
	return fmt.Sprintf("or(%s, %s)", o.left, o.right)
}
