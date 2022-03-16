package tish

import (
	"context"
	"fmt"
	"io"

	"github.com/midbel/tish/internal/words"
)

type Environment interface {
	Resolve(string) ([]string, error)
	Define(string, []string) error
	Delete(string) error
	// SetReadOnly(string)
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
	if e.parent != nil {
		return e.parent.Resolve(ident)
	}
	return nil, fmt.Errorf("%s: undefined variable", ident)
}

func (e *Env) Define(ident string, vs []string) error {
	e.values[ident] = vs
	return nil
}

func (e *Env) Delete(ident string) error {
	delete(e.values, ident)
	return nil
}

type execEnv struct {
	*Shell
}

func getEnvShell(sh *Shell) Environment {
	return execEnv{Shell: sh}
}

func (e execEnv) Execute(ctx context.Context, ex words.Executer, stdout, stderr io.Writer) error {
	sh, err := e.Subshell()
	if err != nil {
		return err
	}
	sh.SetOut(stdout)
	sh.SetErr(stderr)
	return sh.execute(ctx, ex)
}
