package tish

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrReadOnly = errors.New("read only")

type Environment interface {
	Define(string, string) error
	Resolve(string) string
	Delete(string) error
	Environ() []string
}

type variable struct {
	ro   bool
	word string
}

type Env struct {
	parent Environment
	vars   map[string]variable
}

func Environ() Environment {
	vars := make(map[string]variable)
	for _, e := range os.Environ() {
		x := strings.IndexByte(e, '=')
		if x <= 0 {
			continue
		}
		k, v := e[:x], e[x+1:]
		vars[k] = variable{
			ro:   true,
			word: v,
		}
	}
	return &Env{vars: vars}
}

func EmptyEnv() Environment {
	return EnclosedEnv(nil)
}

func EnclosedEnv(env Environment) Environment {
	e := Env{
		parent: env,
		vars:   make(map[string]variable),
	}
	return &e
}

func (e *Env) Define(id string, value string) error {
	v, ok := e.vars[id]
	if ok && v.ro {
		return fmt.Errorf("%s: %w", id, ErrReadOnly)
	}
	e.vars[id] = variable{
		ro:   false,
		word: value,
	}
	return nil
}

func (e *Env) Delete(id string) error {
	v, ok := e.vars[id]
	if !ok && e.parent != nil {
		return e.parent.Delete(id)
	}
	if ok && v.ro {
		return fmt.Errorf("%s: %w", id, ErrReadOnly)
	}
	delete(e.vars, id)
	return nil
}

func (e *Env) Resolve(id string) string {
	w, ok := e.vars[id]
	if !ok && e.parent != nil {
		return e.parent.Resolve(id)
	}
	return w.word
}

func (e *Env) Environ() []string {
	ws := make([]string, 0, len(e.vars))
	for k, v := range e.vars {
		ws = append(ws, fmt.Sprintf("%s=%s", k, v.word))
	}
	if e.parent != nil {
		ps := e.parent.Environ()
		ws = append(ws, ps...)
	}
	return ws
}

func (e *Env) Unwrap() Environment {
	if e.parent != nil {
		return e.parent
	}
	return e
}
