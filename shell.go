package tish

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

var ErrFailed = errors.New("process terminated with failure")

const MaxShellDepth = 100

type Shell struct {
	Args []string

	time.Time
	uid int // user id
	pid int // pid of current shell

	level int // nesting of shell

	globals *Env
	locals  *Env

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	alias map[string]Word

	*Filesystem

	proc struct {
		exit ErrCode // exit code of the last executed process
		pid  int     // pid of the last executed process
	}
}

func DefaultShell() (*Shell, error) {
	fs, errf := DefaultFS()
	if errf != nil {
		return nil, errf
	}
	var (
		in  = Reader(os.Stdin)
		out = Writer(os.Stdout)
		err = Writer(os.Stderr)
	)
	return NewShell(fs, in, out, err), nil
}

func NewShell(fs *Filesystem, in io.Reader, out, err io.Writer) *Shell {
	defer os.Clearenv()

	s := Shell{
		uid:        os.Getuid(),
		pid:        os.Getpid(),
		Time:       time.Now(),
		stdin:      in,
		stdout:     out,
		stderr:     err,
		alias:      make(map[string]Word),
		Filesystem: fs,
	}
	s.globals = NewEnvironment()
	s.locals = NewEnclosedEnvironment(s.globals)

	return &s
}

func (s *Shell) Execute(r io.Reader) error {
	w, err := Parse(r)
	if err != nil {
		return err
	}
	return s.execute(w)
}

func (s *Shell) ExecuteString(str string) error {
	return s.Execute(strings.NewReader(str))
}

func (s *Shell) execute(w Word) error {
	var err error
	switch w := w.(type) {
	case Literal:
		err = s.executeLiteral(w)
	case List:
		err = s.executeList(w)
	case Assignment:
		err = s.executeAssignment(w)
	default:
		err = fmt.Errorf("tish: %T can not be executed", w)
	}
	return err
}

func (s *Shell) executeLiteral(i Literal) error {
	return s.executeSimple([]Word{i})
}

func (s *Shell) executeList(i List) error {
	var execute func([]Word) error
	switch i.kind {
	case kindSeq:
		execute = s.executeSequence
	case kindSimple:
		execute = s.executeSimple
	case kindPipe:
		execute = s.executePipeline
	case kindOr:
		execute = s.executeOr
	case kindAnd:
		execute = s.executeAnd
	case kindShell:
		execute = s.executeSubshell
	default:
		return fmt.Errorf("tish: %s can not be executed", i.kind)
	}
	return execute(i.words)
}

func (s *Shell) executeAssignment(a Assignment) error {
	vs, err := a.Expand(s.locals)
	if err == nil {
		err = s.Define(a.ident, vs)
	}
	return err
}

func (s *Shell) executeSubshell(ws []Word) error {
	return nil
}

func (s *Shell) executeSequence(ws []Word) error {
	var err error
	for _, w := range ws {
		err = s.execute(w)
	}
	return err
}

func (s *Shell) executeOr(ws []Word) error {
	var err error
	for _, w := range ws {
		err = s.execute(w)
		if err == nil && s.proc.exit.Success() {
			break
		}
	}
	return err
}

func (s *Shell) executeAnd(ws []Word) error {
	var err error
	for _, w := range ws {
		err = s.execute(w)
		if err != nil || s.proc.exit.Failure() {
			if err == nil {
				err = ErrFailed
			}
			break
		}
	}
	return err
}

func (s *Shell) executeSubstitution(ws []Word) error {
	var (
		in   bytes.Reader
		out  bytes.Buffer
		err  bytes.Buffer
		word = List{
			kind:  kindSeq,
			words: ws,
		}
	)
	if _, err := s.executeShell(word, &in, &out, &err); err != nil {
		return err
	}
	return nil
}

func (s *Shell) executeShell(w Word, in io.Reader, out, err io.Writer) (ErrCode, error) {
	sh := NewShell(s.Filesystem.Copy(), in, out, err)
	sh.locals = s.locals.Copy()
	sh.globals = sh.locals.Unwrap()
	sh.level = s.level + 1

	errx := sh.execute(w)
	return sh.proc.exit, errx
}

func (s *Shell) executePipeline(ws []Word) error {
	var (
		in  = s.stdin
		err = s.stderr
		grp errgroup.Group
	)
	for i := 0; i < len(ws)-1; i++ {
		p, ok := ws[i].(Pipe)
		if !ok {
			return fmt.Errorf("%s: not a pipe", ws[i])
		}
		r, w := io.Pipe()

		switch p.kind {
		case kindPipe:
			err = s.stderr
		case kindPipeBoth:
			err = w
		default:
			return fmt.Errorf("%s: unexpected pipe type", p.kind)
		}

		grp.Go(func() (errx error) {
			_, errx = s.executeShell(p.Word, in, w, err)
			return
		})

		in = r
	}
	grp.Go(func() (errx error) {
		s.proc.exit, errx = s.executeShell(ws[len(ws)-1], in, s.stdout, s.stderr)
		return
	})
	return grp.Wait()
}

func (s *Shell) executeSimple(ws []Word) error {
	s.Enter()
	defer s.Leave()

	cmd, err := s.buildCommand(ws)
	if err != nil {
		return err
	}
	s.proc.exit = cmd.Run()
	if p, ok := cmd.(interface{ Pid() int }); ok {
		s.proc.pid = p.Pid()
	}
	return nil
}

func (s *Shell) buildCommand(ws []Word) (Command, error) {
	var (
		args = make([]string, 0, len(ws)*4)
		rs   []Redirect
	)

	for _, w := range ws {
		if a, ok := w.(Assignment); ok {
			xs, err := a.Expand(s.locals)
			if err != nil {
				return nil, err
			}
			s.Define(a.ident, xs)
			continue
		}
		if r, ok := w.(Redirect); ok {
			rs = append(rs, r)
			continue
		}
		xs, err := w.Expand(s)
		if err != nil {
			return nil, err
		}
		args = append(args, xs...)
	}

	cmd, err := s.prepare(s.expandFilenames(args))
	if err != nil {
		return nil, err
	}
	for _, r := range rs {
		if r.mode == modRelink {
			switch r.file {
			case fdOut:
				cmd.Copy(fdErr, fdOut) // copy stderr to stdout
			case fdErr:
				cmd.Copy(fdOut, fdErr) // copy stdout to stderr
			default:
				return nil, fmt.Errorf("invalid file descriptor given %d", r.file)
			}
		} else {
			f, err := r.Open(s.locals)
			if err != nil {
				return nil, err
			}
			if err := cmd.Replace(r.file, f); err != nil {
				return nil, err
			}
		}
	}
	return cmd, nil
}

func (s *Shell) prepare(args []string) (Command, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command given")
	}
	s.proc.pid = 0
	s.proc.exit = ExitOk

	if w, ok := s.alias[args[0]]; ok {
		vs, err := w.Expand(s.locals)
		if err != nil {
			return nil, err
		}
		args = append(vs, args[1:]...)
	}

	if c, ok := builtins[args[0]]; ok && c.Runnable() {
		c.Stdin = s.stdin
		c.Stdout = s.stdout
		c.Stderr = s.stderr

		c.Shell = s
		c.Args = args[1:]

		return &c, nil
	}
	c := exec.Command(args[0], args[1:]...)

	if es := s.locals.Environ(); len(es) > 0 {
		c.Env = append(c.Env, es...)
	}
	c.Dir = s.Cwd()

	c.Stdin = s.stdin
	c.Stdout = s.stdout
	c.Stderr = s.stderr

	return &Cmd{Cmd: c}, nil
}

func (s *Shell) expandFilenames(args []string) []string {
	// for _, a := range args {
	//
	// }
	return args
}

func (s *Shell) RegisterAlias(ident, alias string) error {
	w, err := Parse(strings.NewReader(alias))
	if err != nil {
		return err
	}
	s.alias[ident] = w
	return nil
}

func (s *Shell) UnregisterAlias(alias string) {
	if alias == "" {
		s.alias = make(map[string]Word)
	} else {
		delete(s.alias, alias)
	}
}

func (s *Shell) Exit(n ErrCode) {
	os.Exit(int(n))
}

func (s *Shell) Enter() {
	s.locals = NewEnclosedEnvironment(s.locals)
}

func (s *Shell) Leave() {
	s.locals.Unwrap()
}

func (s *Shell) Resolve(ident string) ([]string, error) {
	var (
		vs  = make([]string, 0, 1)
		err error
	)
	switch ident {
	case "?":
		if s.proc.exit > 0 {
			vs = append(vs, strconv.Itoa(int(s.proc.exit)))
		}
	case "$":
		vs = append(vs, strconv.Itoa(s.pid))
	case "!":
		if s.proc.pid > 0 {
			vs = append(vs, strconv.Itoa(s.proc.pid))
		}
	case "#":
		vs = append(vs, strconv.Itoa(len(s.Args)))
	case "@":
		vs = append(vs, s.Args...)
	case "UID":
		vs = append(vs, strconv.Itoa(int(s.uid)))
	case "PWD":
		vs = append(vs, s.Cwd())
	case "VERSION":
		vs = append(vs, Version)
	case "HOSTNAME":
		if h, err := os.Hostname(); err == nil {
			vs = append(vs, h)
		}
	case "OS":
		vs = append(vs, runtime.GOOS)
	case "SHELL":
		vs = append(vs, Tish)
	case "SECONDS":
		elapsed := time.Now().UTC().Sub(s.Time)
		vs = append(vs, strconv.Itoa(int(elapsed.Seconds())))
	default:
		vs, err = s.locals.Resolve(ident)
	}
	return vs, err
}

func (s *Shell) Define(ident string, values []string) error {
	err := s.locals.Define(ident, values)
	if err != nil && !errors.Is(err, ErrReadOnly) {
		err = nil
	}
	return err
}

func (s *Shell) Export(ident string, values []string) {
	s.globals.Define(ident, values)
}

func (s *Shell) SetReadOnly(ident string, ro bool) {
	s.locals.SetReadOnly(ident, ro)
}

func (s *Shell) Environ() []string {
	return s.locals.Environ()
}

func sameFile(in, out interface{}) error {
	fin, ok := in.(*os.File)
	if !ok {
		return nil
	}
	fout, ok := out.(*os.File)
	if !ok {
		return nil
	}
	var err error
	if fin.Name() == fout.Name() {
		err = fmt.Errorf("%s: already open for reading", fin.Name())
	}
	return err
}

func closeFile(c interface{}) {
	if c, ok := c.(io.Closer); ok {
		c.Close()
	}
}

type shellWriter struct {
	inner io.Writer
}

func Writer(w io.Writer) io.Writer {
	if _, ok := w.(io.Closer); !ok {
		return w
	}
	if _, ok := w.(*shellWriter); ok {
		return w
	}
	return &shellWriter{inner: w}
}

func (w *shellWriter) Write(bs []byte) (int, error) {
	return w.inner.Write(bs)
}

type shellReader struct {
	inner io.Reader
}

func Reader(r io.Reader) io.Reader {
	if _, ok := r.(io.Closer); !ok {
		return r
	}
	if _, ok := r.(*shellReader); ok {
		return r
	}
	return &shellReader{inner: r}
}

func (r *shellReader) Read(bs []byte) (int, error) {
	return r.inner.Read(bs)
}
