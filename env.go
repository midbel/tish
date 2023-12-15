package tish

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	ErrDefined   = errors.New("variable not defined")
	ErrReadOnly  = errors.New("read only variable")
	ErrImmutable = errors.New("immutable env")
)

type Environment interface {
	Define(string, []string) error
	Resolve(string) ([]string, error)
}

type ImmutableEnv struct {
	Environment
}

func Immutable(env Environment) Environment {
	return ImmutableEnv{
		Environment: env,
	}
}

func (e *ImmutableEnv) Define(_ string, _ []string) error {
	return ErrImmutable
}

type Env struct {
	parent Environment
	values map[string]variable
}

func EmptyEnv() Environment {
	return EnclosedEnv(nil)
}

func EnclosedEnv(parent Environment) Environment {
	return &Env{
		parent: parent,
		values: make(map[string]variable),
	}
}

func (e *Env) Resolve(ident string) ([]string, error) {
	vs, ok := e.values[ident]
	if ok {
		return slices.Clone(vs.values), nil
	}
	if e.parent == nil {
		return nil, undefined(ident)
	}
	return e.parent.Resolve(ident)
}

func (e *Env) Define(ident string, values []string) error {
	vs, ok := e.values[ident]
	if !ok {
		e.values[ident] = variable{
			ro:     false,
			values: slices.Clone(values),
		}
		return nil
	}
	if vs.ro {
		return readonly(ident)
	}
	vs.values = slices.Clone(values)
	e.values[ident] = vs
	return nil
}

func (e *Env) List() []string {
	var list []string
	if i, ok := e.parent.(interface{ List() []string }); ok {
		list = i.List()
	}
	for k, vs := range e.values {
		kv := fmt.Sprintf("%s=%s", k, strings.Join(vs.values, " "))
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

func readonly(ident string) error {
	return fmt.Errorf("%s: %w", ident, ErrReadOnly)
}

type variable struct {
	ro     bool
	values []string
}
