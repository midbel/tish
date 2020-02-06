package tish

import (
	"fmt"
)

type Env struct {
	parent *Env
	locals map[string][]string
}

func NewEnvironment() *Env {
	return NewEnclosedEnvironment(nil)
}

func NewEnclosedEnvironment(e *Env) *Env {
	return &Env{
		locals: make(map[string][]string),
		parent: e,
	}
}

func (e *Env) Get(n string) ([]string, error) {
	vs, ok := e.locals[n]
	if ok {
		return vs, nil
	}
	if e.parent != nil {
		return e.parent.Get(n)
	}
	return nil, fmt.Errorf("%s: not defined", n)
}

func (e *Env) Set(n string, vs []string) {
	e.locals[n] = vs
}

func (e *Env) Del(n string) {
	if _, ok := e.locals[n]; !ok && e.parent != nil {
		e.parent.Del(n)
		return
	}
	delete(e.locals, n)
}
