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
	stdin  redirect
	stdout redirect
	stderr redirect
}

func (c command) Exec(e *Env) error {
	_, err := c.word.Expand(e)
	if err != nil {
		return err
	}
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

type mode byte

const (
  modeReadOnly mode = iota
  modeCreate
  modeAppend
)

type redirect struct {
  word Word
  mode mode
}

func (r redirect) Open(e *Env) (*os.File, error) {
  vs, err := r.word.Expand(e)
  if err != nil {
    return nil, err
  }
  if len(vs) == 0 {
    return nil, fmt.Errorf("")
  }
  var flag int
  switch r.mode {
  case modeReadOnly:
    flag = os.O_RDONLY
  case modeCreate:
    flag = os.O_CREATE|os.O_TRUNC|os.O_WRONLY
  case modeAppend:
    flag = os.O_APPEND|os.O_CREATE|os.O_WRONLY
  default:
    return nil, fmt.Errorf("unsupported mode")
  }
  return os.OpenFile(vs[0], flag, 0644)
}
