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
	// SetOut(io.Writer)
	// SetErr(io.Writer)
	Execute(context.Context, Executer, io.Writer, io.Writer) error
}
