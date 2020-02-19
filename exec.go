package tish

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

const MaxHistSize = 100

var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
	stdin  io.Reader = os.Stdin
)

type Shell struct {
	time.Time
	uid int // user id
	pid int // pid of current shell

	level int // nesting of shell

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
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
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
	sh := NewShell()
	sh.level = s.level + 1
	return sh
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
		err = executePipeline(i.words, e)
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

func executePipeline(ws []Word, e *Env) error {
  var (
    pr io.Reader
    n  = len(ws)-1
    cs = make([]Command, len(ws))
  )
	for i := 0; i < len(ws); i++ {
		w := ws[i]
		args, err := w.Expand(e)
		if err != nil || len(args) == 0 {
			return err
		}
    var cmd Command
    if i == 0 {
      r, w := io.Pipe()
      cmd, pr = prepare(args, stdin, w, stderr), r
    } else if i == n {
      cmd = prepare(args, pr, stdout, stderr)
    } else {
      r, w := io.Pipe()
      cmd, pr = prepare(args, pr, w, stderr), r
    }
    cs[i] = cmd
	}
  for i := 0; i < n; i++ {
    if err := cs[i].Start(); err != nil {
      return err
    }
  }
	return cs[n].Run()
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
	return prepare(vs, stdin, stdout, stderr).Run()
}

func executeLiteral(i Literal, e *Env) error {
	vs, err := i.Expand(e)
	if err != nil || len(vs) == 0 {
		return err
	}
	return prepare(vs, stdin, stdout, stderr).Run()
}

func prepare(args []string, in io.Reader, out, err io.Writer) Command {
	if c, ok := builtins[args[0]]; ok && c.Runnable() {
		c.args = args[1:]
		c.stdin = in
		c.stdout = out
		c.stderr = err

		return &c
	}
	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = err

	return cmd
}
