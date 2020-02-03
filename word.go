package tish

import (
  "fmt"
  "strings"
)

type Word interface {
	Expand(*Env) ([]string, error)
	fmt.Stringer
}

type List struct {
  words []Word
  kind  rune
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
}

func (i List) String() string {
  var buf strings.Builder
  for _, w := range i.words {
    buf.WriteString(w.String())
  }
  return buf.String()
}

type Variable string

func (v Variable) Expand(e *Env) ([]string, error) {
  return nil, nil
}

func (v Variable) String() string {
  return "$"+string(v)
}

type Literal string

func (i Literal) Expand(_ *Env) ([]string, error) {
  return []string{string(i)}, nil
}
