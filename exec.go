package tish

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

const MaxHistSize = 100

type Shell struct {
	time.Time
	uid int // user id
	pid int // pid of current shell

	depth int // nesting of shell

	globals *Env
	locals  *Env

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	alias map[string]string

	dirs struct {
		ptr  int
		hist []string
	}
	proc struct {
		exit int // exit code of the last executed process
		pid  int // pid of the last executed process
	}
}

func NewShell() *Shell {
	s := Shell{
		uid:     os.Getuid(),
		pid:     os.Getpid(),
		Time:    time.Now(),
		globals: NewEnvironment(),
		locals:  NewEnvironment(),
		stdin:   os.Stdin,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
		alias:   make(map[string]string),
	}
	s.dirs.hist = make([]string, MaxHistSize)
	return &s
}

func (s *Shell) workingDir() string {
	return s.dirs.hist[s.dirs.ptr-1]
}

func (s *Shell) popDir() {
	s.dirs.ptr--
}

func (s *Shell) pushDir(dir string) {
	s.dirs.hist[s.dirs.ptr] = dir
	s.dirs.ptr++
}

func (s *Shell) subshell() *Shell {
	return nil
}

func Execute(r io.Reader) error {
	return ExecuteWithEnv(r, NewEnvironment())
}

func ExecuteWithEnv(r io.Reader, env *Env) error {
	ws, err := Parse(r)
	if err != nil {
		return err
	}
	env = NewEnclosedEnvironment(env)
	switch w := ws.(type) {
	case Literal:
		err = executeLiteral(w, env)
	case List:
		err = executeList(w, env)
	default:
		err = fmt.Errorf("exec: %T can not be executed", w)
	}
	return err
}

func executeList(i List, e *Env) error {
	var err error
	switch i.kind {
	case kindSimple:
		err = executeSimple(i, e)
	case kindPipe:
		err = executePipeline(i, e)
	case kindSeq:
		err = executeSequence(i.words, e)
	case kindAnd:
		err = executeAnd(i.words, e)
	case kindOr:
		err = executeOr(i.words, e)
	default:
		err = fmt.Errorf("exec: %s can not be executed", i.kind)
	}
	return err
}

func executeOr(ws []Word, e *Env) error {
	var err error
	for _, w := range ws {
		switch w := w.(type) {
		case Literal:
			err = executeLiteral(w, e)
		case List:
			err = executeList(w, e)
		default:
			return fmt.Errorf("exec: %T can not be executed", w)
		}
		if err == nil {
			break
		}
	}
	return err
}

func executeAnd(ws []Word, e *Env) error {
	var err error
	for _, w := range ws {
		switch w := w.(type) {
		case Literal:
			err = executeLiteral(w, e)
		case List:
			err = executeList(w, e)
		default:
			return fmt.Errorf("exec: %T can not be executed", w)
		}
		if err != nil {
			break
		}
	}
	return err
}

func executePipeline(i List, e *Env) error {
	return nil
}

func executeSequence(ws []Word, e *Env) error {
	var err error
	for _, w := range ws {
		switch w := w.(type) {
		case Literal:
			err = executeLiteral(w, e)
		case List:
			err = executeList(w, e)
		default:
			return fmt.Errorf("exec: %T can not be executed", w)
		}
	}
	return err
}

func executeSimple(w Word, e *Env) error {
	vs, err := w.Expand(e)
	if err != nil || len(vs) == 0 {
		return err
	}
	if c, ok := builtins[vs[0]]; ok && c.Runnable() {
		return c.Run(c, vs[1:])
	}
	return prepare(vs, os.Stdin, os.Stdout, os.Stderr).Run()
}

func executeLiteral(i Literal, e *Env) error {
	vs, err := i.Expand(e)
	if err != nil || len(vs) == 0 {
		return err
	}
	if c, ok := builtins[vs[0]]; ok && c.Runnable() {
		return c.Run(c, vs[1:])
	}
	return prepare(vs, os.Stdin, os.Stdout, os.Stderr).Run()
}

func prepare(args []string, in io.Reader, out, err io.Writer) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = err

	return cmd
}
