package tish

import (
	"fmt"
)

type variable struct {
	ro   bool
	word Word
}

type Env struct {
	parent *Env
	vars   map[string]variable
}

func EmptyEnv() *Env {
	return EnclosedEnv(nil)
}

func EnclosedEnv(env *Env) *Env {
	e := Env{
		parent: env,
		vars:   make(map[string]variable),
	}
	return &e
}

func (e *Env) Define(id string, value Word) {
	v, ok := e.vars[id]
	if ok && v.ro {
		return
	}
	e.vars[id] = variable{
		ro:   false,
		word: value,
	}
}

func (e *Env) Delete(id string) {
	_, ok := e.vars[id]
	if !ok && e.parent != nil {
		e.parent.Delete(id)
		return
	}
	if ok {
		delete(e.vars, id)
	}
}

func (e *Env) Resolve(id string) Word {
	w, ok := e.vars[id]
	if !ok && e.parent != nil {
		return e.parent.Resolve(id)
	}
	return w.word
}

func (e *Env) Environ() []string {
	ws := make([]string, 0, len(e.vars))
	for k, v := range e.vars {
		ws = append(ws, fmt.Sprintf("%s=%s", k, v.word.Expand(e)))
	}
	if e.parent != nil {
		ps := e.parent.Environ()
		ws = append(ws, ps...)
	}
	return ws
}
