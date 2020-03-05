package tish

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrReadOnly   = errors.New("variable is read only")
	ErrNotDefined = errors.New("variable not defined")
)

type Environment interface {
	Resolve(string) ([]string, error)
	Define(string, []string) error
	SetReadOnly(string, bool)

	Environ() []string
	LocalEnviron() []string
}

type envval struct {
	Values   []string
	ReadOnly bool
}

type Env struct {
	parent *Env
	locals map[string]envval
}

func NewEnvironment() *Env {
	return NewEnclosedEnvironment(nil)
}

func NewEnclosedEnvironment(e *Env) *Env {
	return &Env{
		locals: make(map[string]envval),
		parent: e,
	}
}

func (e *Env) Resolve(n string) ([]string, error) {
	return e.resolve(n)
}

func (e *Env) Define(n string, vs []string) error {
	return e.define(n, vs)
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

func (e *Env) SetReadOnly(ident string, ro bool) {
	ev, ok := e.locals[ident]
	if !ok && e.parent != nil {
		e.parent.SetReadOnly(ident, ro)
		return
	}
	ev.ReadOnly = ro
	e.locals[ident] = ev
}

func (e *Env) LocalEnviron() []string {
	var env []string
	for k, vs := range e.locals {
		str := fmt.Sprintf("%s=%s", k, strings.Join(vs.Values, " "))
		env = append(env, str)
	}
	return env
}

func (e *Env) Environ() []string {
	var env []string
	if e.parent != nil {
		env = e.parent.Environ()
	}
	return append(env, e.LocalEnviron()...)
}

func (e *Env) Unwrap() *Env {
	if e.parent == nil {
		return e
	}
	return e.parent
}

func (e *Env) Copy() *Env {
	vs := make(map[string]envval)
	for k, ev := range e.locals {
		e := envval{
			ReadOnly: ev.ReadOnly,
			Values:   make([]string, len(ev.Values)),
		}
		copy(e.Values, ev.Values)
		vs[k] = e
	}
	return &Env{locals: vs}
}

func (e *Env) resolve(n string) ([]string, error) {
	val, ok := e.locals[n]
	if ok {
		vs := make([]string, len(val.Values))
		copy(vs, val.Values)
		return vs, nil
	}
	if e.parent != nil {
		return e.parent.resolve(n)
	}
	return nil, fmt.Errorf("%s: %w", n, ErrNotDefined)
}

func (e *Env) define(n string, vs []string) error {
	val, ok := e.locals[n]
	if ok && val.ReadOnly {
		return fmt.Errorf("%s: %w", n, ErrReadOnly)
	}
	e.locals[n] = envval{
		Values:   vs,
		ReadOnly: false,
	}
	return nil
}
