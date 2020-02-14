package tish

import (
	"os/exec"
)

type Command interface {
	Execute(*Env) error
}

type command struct {
	word Word
}

func (c command) Execute(e *Env) int {
	vs, err := c.word.Expand(e)
	if err != nil || len(vs) == 0 {
		if err == nil {
			err = fmt.Errorf("")
		}
		return err
	}
	cmd := exec.Command(vs[0], vs[1:]...)
	// bind cmd woring dir, env, stdout, stderr, stdin
	return cmd.Run()
}

type sequence []Command

func (es sequence) Execute(e *Env) error {
	var err error
	for _, c := range es {
		err = c.Execute(e)
	}
	return err
}

type pipeline []Command

func (p pipeline) Execute(e *Env) error {
	return nil
}
