package words

import (
	"context"
	"io"
)

type Environment interface {
	Resolve(string) ([]string, error)
	Define(string, []string) error
}

type ShellEnv interface {
	Environment
	SetOut(io.Writer)
	Execute(context.Context, Executer) error
}
