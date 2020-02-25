package tish

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

const MaxHistSize = 100

// var (
// 	stdout io.Writer = os.NewFile(os.Stdout.Fd(), "in")
// 	stderr io.Writer = os.NewFile(os.Stderr.Fd(), "out")
// 	stdin  io.Reader = os.NewFile(os.Stdin.Fd(), "err")
// )

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
	defer os.Clearenv()
	s := Shell{
		uid:     os.Getuid(),
		pid:     os.Getpid(),
		Time:    time.Now(),
		globals: NewEnvironment(),
		stdin:   os.NewFile(os.Stdin.Fd(), "stdin"),
		stdout:  os.NewFile(os.Stdout.Fd(), "stdout"),
		stderr:  os.NewFile(os.Stderr.Fd(), "stderr"),
		alias:   make(map[string]string),
	}
	s.locals = NewEnclosedEnvironment(s.globals)
	s.dirs.hist = make([]string, MaxHistSize)
	return &s
}

func (s *Shell) Exit() {
	s.exit(0)
}

func (s *Shell) exit(n int) {
	os.Exit(n)
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
	case Assignment:
		err = executeAssignment(w, env)
	case List:
		err = executeList(w, env)
	default:
		err = fmt.Errorf("exec: %T can not be executed", w)
	}
	return err
}

func executeAssignment(a Assignment, e *Env) error {
	_, err := a.Expand(e)
	return err
}

func executeList(i List, e *Env) error {
	var err error
	switch i.kind {
	case kindSimple:
		err = executeSimple(i.words, e)
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
	in := os.NewFile(os.Stdin.Fd(), "stdin")
	for i := 0; ; i++ {
		args, err := ws[i].Expand(e)
		if err != nil || len(args) == 0 {
			return err
		}
		if i == len(ws)-1 {
			var (
				out = os.NewFile(os.Stdout.Fd(), "stdout")
				err = os.NewFile(os.Stderr.Fd(), "stderr")
			)
			return prepare(args, e, in, out, err).Run()
		}
		p, ok := ws[i].(Pipe)
		if !ok {
			return fmt.Errorf("%s: not a pipe", ws[i])
		}
		r, w, err := os.Pipe()
		if err != nil {
			return err
		}
		var serr io.Writer
		switch p.kind {
		case kindPipe:
			serr = os.NewFile(os.Stderr.Fd(), "stderr")
		case kindPipeBoth:
			serr = w
		default:
			return fmt.Errorf("%s: unexpected pipe type", p.kind)
		}
		if err := prepare(args, e, in, w, serr).Start(); err != nil {
			return err
		}
		in = r
	}
}

func executeSequence(ws []Word, e *Env) error {
	var err error
	for _, w := range ws {
		switch w := w.(type) {
		case Literal:
			err = executeLiteral(w, e)
		case List:
			err = executeList(w, e)
		case Assignment:
			err = executeAssignment(w, e)
		default:
			return fmt.Errorf("exec: %T can not be executed", w)
		}
	}
	return err
}

func executeSimple(ws []Word, e *Env) error {
	var (
		rs   []Redirect
		args []string
		env  = NewEnclosedEnvironment(e)
	)
	for _, w := range ws {
		if r, ok := w.(Redirect); ok {
			rs = append(rs, r)
			continue
		}
		if a, ok := w.(Assignment); ok {
			if _, err := a.Expand(env); err != nil {
				return err
			}
			continue
		}
		xs, err := w.Expand(env)
		if err != nil {
			return err
		}
		args = append(args, xs...)
	}
	var (
		in  = os.NewFile(os.Stdin.Fd(), "stdin")
		out = os.NewFile(os.Stdout.Fd(), "stdout")
		err = os.NewFile(os.Stderr.Fd(), "stderr")
	)

	for _, r := range rs {
		f, errf := r.Open(e)
		if errf != nil {
			return errf
		}

		if in.Name() == f.Name() {
			f.Close()
			return fmt.Errorf("%s: already open for reading", f.Name())
		}

		switch r.file {
		case fdIn:
			in.Close()
			in = f
		case fdOut:
			out.Close()
			out = f
		case fdErr:
			err.Close()
			err = f
		case fdBoth:
			out.Close()
			err.Close()
			out, err = f, f
		default:
			return fmt.Errorf("invalid file descriptor %d", r.file)
		}
	}
	return prepare(args, env, in, out, err).Run()
}

func executeLiteral(i Literal, e *Env) error {
	vs, errf := i.Expand(e)
	if errf != nil || len(vs) == 0 {
		return errf
	}
	var (
		in  = os.NewFile(os.Stdin.Fd(), "stdin")
		out = os.NewFile(os.Stdout.Fd(), "stdout")
		err = os.NewFile(os.Stderr.Fd(), "stderr")
	)
	return prepare(vs, e, in, out, err).Run()
}

func prepare(args []string, env *Env, in io.Reader, out, err io.Writer) Command {
	if c, ok := builtins[args[0]]; ok && c.Runnable() {
		c.stdin = in
		c.stdout = out
		c.stderr = err

		c.Args = args[1:]
		if env == nil {
			env = NewEnvironment()
		}
		c.Env = env

		return &c
	}
	cmd := exec.Command(args[0], args[1:]...)

	if es := env.Values(); len(es) > 0 {
		cmd.Env = append(cmd.Env, es...)
	}
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = err

	return cmd
}
