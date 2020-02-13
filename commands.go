package tish

import (
  "fmt"
  "os"
)

type Command interface {
	Exec(*Env) error
}

type command struct {
	word   Word
}

func (c command) Exec(e *Env) error {
	return nil
}

type sequence []Command

func (s sequence) Exec(e *Env) error {
	return nil
}

type pipeline []Command

func (p pipeline) Exec(e *Env) error {
	return nil
}
