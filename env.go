package tish

import (
	"fmt"
	"strings"
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

func (e *Env) Clear() {
	for k := range e.locals {
		delete(e.locals, k)
	}
}

func (e *Env) Values() []string {
	var env []string
	if e.parent != nil {
		env = e.parent.Values()
	}
	for k, vs := range e.locals {
		str := fmt.Sprintf("%s=%s", k, strings.Join(vs, " "))
		env = append(env, str)
	}
	return env
}

func (e *Env) Unwrap() (*Env, error) {
	if e.parent == nil {
		return nil, fmt.Errorf("env: can not unwrap globals")
	}
	return e.parent, nil
}
