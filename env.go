package tish

import (
	"errors"
	"fmt"
)

var ErrDefined = errors.New("variable not defined")

type Environment interface {
	Define(string, []string)
	Resolve(string) ([]string, error)
}

type Env struct {
	parent Environment
	values map[string][]string
}

func EmptyEnv() Environment {
	return EnclosedEnv(nil)
}

func EnclosedEnv(parent Environment) Environment {
	return &Env{
		parent: parent,
		values: make(map[string][]string),
	}
}

func (e *Env) Resolve(ident string) ([]string, error) {
	vs, ok := e.values[ident]
	if ok {
		return vs, nil
	}
	if e.parent == nil {
		return nil, undefined(ident)
	}
	return e.parent.Resolve(ident)
}

func (e *Env) Define(ident string, values []string) {
	e.values[ident] = append(e.values[ident][:0], values...)
}

func (e *Env) List() []string {
	var list []string
	if i, ok := e.parent.(interface{ List() []string }); ok {
		list = i.List()
	}
	for k, vs := range e.values {
		kv := fmt.Sprintf("%s=%s", k, strings.Join(vs, " "))
		list = append(list, kv)
	}
	return list
}

func (e *Env) unwrap() Environment {
	if e.parent == nil {
		return e
	}
	return e.parent
}

func undefined(ident string) error {
	return fmt.Errorf("%s: %w", ident, ErrDefined)
}
