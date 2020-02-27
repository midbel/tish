package tish

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	ErrFailed = errors.New("process terminated with failure")
	ErrDry    = errors.New("dry run")
)

var DefaultProfile string

const (
	MaxHistSize = 100
	Version     = "0.0.1"
	Tish        = "tish"
)

func init() {
	home, _ := os.UserHomeDir()
	DefaultProfile = filepath.Join(home, ".tishrc")
}

type ErrCode int

func (e ErrCode) Success() bool {
	return e == ExitOk
}

func (e ErrCode) Failure() bool {
	return e != ExitOk
}

const (
	ExitOk ErrCode = iota
	ExitHelp
	ExitBadUsage
	ExitIO
	ExitVariable
	ExitExec
	ExitNotExec
	ExitDoneExec
	ExitUnknown
	ExitNoFile
)

type Command interface {
	Start() error
	Wait() ErrCode
	Run() ErrCode

	Replace(int, *os.File) error
	Copy(int, int)
}

type Shell struct {
	Dry     bool
	Verbose bool
	Args    []string

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

	dirs struct {
		ptr  int
		hist []string
	}
	proc struct {
		exit ErrCode // exit code of the last executed process
		pid  int     // pid of the last executed process
	}
}

func DefaultShell() *Shell {
	// var (
	// 	in  = os.NewFile(os.Stdin.Fd(), "stdin")
	// 	out = os.NewFile(os.Stdout.Fd(), "stdout")
	// 	err = os.NewFile(os.Stderr.Fd(), "stderr")
	// )
	return NewShell(os.Stdin, os.Stdout, os.Stderr)
}

func NewShell(in io.Reader, out, err io.Writer) *Shell {
	defer os.Clearenv()
	s := Shell{
		uid:    os.Getuid(),
		pid:    os.Getpid(),
		Time:   time.Now(),
		stdin:  in,
		stdout: out,
		stderr: err,
		alias:  make(map[string]Word),
	}
	s.globals = NewEnvironment()
	s.locals = NewEnclosedEnvironment(s.globals)

	s.dirs.hist = make([]string, MaxHistSize)

	if cwd, err := os.Getwd(); err == nil {
		s.PushDir(cwd)
	}

	return &s
}

func (s *Shell) Enter() {
	s.locals = NewEnclosedEnvironment(s.locals)
}

func (s *Shell) Leave() {
	s.locals.Unwrap()
}

func (s *Shell) Subshell() *Shell {
	sh := DefaultShell()
	sh.level = s.level + 1
	return sh
}

func (s *Shell) Cwd() string {
	return s.dirs.hist[s.dirs.ptr-1]
}

func (s *Shell) PushDir(dir string) error {
	i, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !i.IsDir() {
		return fmt.Errorf("%s: not a directory", dir)
	}
	ix := s.dirs.ptr % MaxHistSize
	s.dirs.hist[ix] = dir
	s.dirs.ptr++

	return nil
}

func (s *Shell) PopDir() {
	s.dirs.ptr--
	if s.dirs.ptr < 0 {
		s.dirs.ptr = MaxHistSize - 1
	}
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
	if errors.Is(err, ErrDry) {
		err = nil
	}
	return err
}

func (s *Shell) executeLiteral(i Literal) error {
	s.Enter()
	defer s.Leave()

	args, err := i.Expand(s.locals)
	if err != nil {
		return err
	}
	cmd, err := s.prepare(args)
	if err != nil {
		return err
	}
	s.proc.exit = cmd.Run()
	if p, ok := cmd.(interface{ Pid() int }); ok {
		s.proc.pid = p.Pid()
	}
	return nil
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

func (s *Shell) executePipeline(ws []Word) error {
	var in *os.File
	for i := 0; ; i++ {
		args, err := ws[i].Expand(s.locals)
		if err != nil || len(args) == 0 {
			return err
		}
		cmd, err := s.prepare(args)
		if err != nil {
			return err
		}
		if i == len(ws)-1 {
			cmd.Replace(fdIn, in)
			s.proc.exit = cmd.Run()
			if p, ok := cmd.(interface{ Pid() int }); ok {
				s.proc.pid = p.Pid()
			}
			break
		}
		p, ok := ws[i].(Pipe)
		if !ok {
			return fmt.Errorf("%s: not a pipe", ws[i])
		}
		r, w, err := os.Pipe()
		if err != nil {
			return err
		}
		if i > 0 {
			cmd.Replace(fdIn, in)
		}
		cmd.Replace(fdOut, w)
		switch p.kind {
		case kindPipe:
		case kindPipeBoth:
			cmd.Replace(fdErr, w)
		default:
			return fmt.Errorf("%s: unexpected pipe type", p.kind)
		}
		if err := cmd.Start(); err != nil {
			return err
		}
		in = r
	}
	return nil
}

func (s *Shell) executeSimple(ws []Word) error {
	s.Enter()
	defer s.Leave()

	var (
		args = make([]string, 0, len(ws)*4)
		rs   []Redirect
	)

	for _, w := range ws {
		if a, ok := w.(Assignment); ok {
			xs, err := a.Expand(s.locals)
			if err != nil {
				return err
			}
			s.Define(a.ident, xs)
			continue
		}
		if r, ok := w.(Redirect); ok {
			rs = append(rs, r)
			continue
		}
		xs, err := w.Expand(s.locals)
		if err != nil {
			return err
		}
		args = append(args, xs...)
	}

	cmd, err := s.prepare(args)
	if err != nil {
		return err
	}
	for _, r := range rs {
		if r.mode == modRelink {
			switch r.file {
			case fdOut:
				cmd.Copy(fdErr, fdOut) // copy stderr to stdout
			case fdErr:
				cmd.Copy(fdOut, fdErr) // copy stdout to stderr
			default:
				return fmt.Errorf("invalid file descriptor given %d", r.file)
			}
		} else {
			f, err := r.Open(s.locals)
			if err != nil {
				return err
			}
			if err := cmd.Replace(r.file, f); err != nil {
				return err
			}
		}
	}
	s.proc.exit = cmd.Run()
	if p, ok := cmd.(interface{ Pid() int }); ok {
		s.proc.pid = p.Pid()
	}
	return nil
}

func (s *Shell) executeSubstitution(w Word) (Word, error) {
	var (
		sin  = bytes.NewReader(nil)
		sout bytes.Buffer
		serr bytes.Buffer
		sh   = NewShell(sin, &sout, &serr)
	)
	if err := sh.execute(w); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Shell) expandWord(ws []Word) ([]string, error) {
	var (
		args []string
		err  error
	)
	for _, w := range ws {
		var xs []string
		switch x := w.(type) {
		case List:
			if x.kind == kindSub {
				w, err = s.executeSubstitution(w)
				if err != nil {
					return nil, err
				}
			}
			xs, err = w.Expand(s.locals)
		default:
			xs, err = w.Expand(s.locals)
		}
		if err != nil {
			break
		}
		args = append(args, xs...)
	}
	return args, err
}

func (s *Shell) prepare(args []string) (Command, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command given")
	}
	s.proc.pid = 0
	s.proc.exit = ExitOk

	in := s.stdin // s.duplicateReader(s.stdin)
	out := s.stdout // s.duplicateWriter(s.stdout)
	err := s.stderr // s.duplicateWriter(s.stderr)

	if s.Verbose {
		fmt.Fprintln(s.stderr, strings.Join(args, " "))
	}
	if s.Dry {
		return nil, ErrDry
	}

	if w, ok := s.alias[args[0]]; ok {
		vs, err := w.Expand(s.locals)
		if err != nil {
			return nil, err
		}
		args = append(vs, args[1:]...)
	}

	if c, ok := builtins[args[0]]; ok && c.Runnable() {
		c.Stdin = in
		c.Stdout = out
		c.Stderr = err

		c.Shell = s
		c.Args = args[1:]

		return &c, nil
	}
	c := exec.Command(args[0], args[1:]...)

	if es := s.locals.Environ(); len(es) > 0 {
		c.Env = append(c.Env, es...)
	}
	c.Dir = s.Cwd()
	c.Stdin = in
	c.Stdout = out
	c.Stderr = err

	return &Cmd{Cmd: c}, nil
}

func (s *Shell) duplicateReader(fd io.Reader) io.Reader {
	if f, ok := fd.(*os.File); ok {
		return os.NewFile(f.Fd(), f.Name())
	}
	return fd
}

func (s *Shell) duplicateWriter(fd io.Writer) io.Writer {
	if f, ok := fd.(*os.File); ok {
		return os.NewFile(f.Fd(), f.Name())
	}
	return fd
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

func (s *Shell) Resolve(ident string) []string {
	vs := make([]string, 0, 1)
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
	case "HOME":
	case "SHELL":
		vs = append(vs, Tish)
	case "SECONDS":
		elapsed := time.Now().UTC().Sub(s.Time)
		vs = append(vs, strconv.Itoa(int(elapsed.Seconds())))
	default:
		vs, _ = s.locals.Get(ident)
	}
	return vs
}

func (s *Shell) Define(ident string, values []string) error {
	err := s.locals.Set(ident, values)
	if err != nil && !errors.Is(err, ErrReadOnly) {
		err = nil
	}
	return err
}

func (s *Shell) Export(ident string, values []string) {
	s.globals.Set(ident, values)
}

func (s *Shell) SetReadOnly(ident string, ro bool) {
	s.locals.SetReadOnly(ident, ro)
}

func (s *Shell) Environ() []string {
	return s.locals.Environ()
}

type Cmd struct {
	*exec.Cmd
}

func (c *Cmd) Copy(src, dst int) {
	if src == dst {
		return
	}
	switch src {
	case fdOut:
		c.Stderr = c.Stdout
	case fdErr:
		c.Stdout = c.Stderr
	default:
	}
}

func (c *Cmd) Replace(fd int, f *os.File) error {
	switch fd {
	case fdIn:
		// closeFile(c.Stdin)
		c.Stdin = f
	case fdOut:
		if err := sameFile(c.Stdin, f); err != nil {
			return err
		}
		// closeFile(c.Stdout)
		c.Stdout = f
	case fdErr:
		if err := sameFile(c.Stdin, f); err != nil {
			return err
		}
		// closeFile(c.Stderr)
		c.Stderr = f
	case fdBoth:
		if err := sameFile(c.Stdin, f); err != nil {
			return err
		}
		// closeFile(c.Stdout)
		// closeFile(c.Stderr)
		c.Stdout, c.Stderr = f, f
	default:
		return fmt.Errorf("invalid file description %d", fd)
	}
	return nil
}

func (c *Cmd) Pid() int {
	return c.ProcessState.Pid()
}

func (c *Cmd) Wait() ErrCode {
	var (
		code ErrCode
		exit *exec.ExitError
		err  = c.Cmd.Wait()
	)
	if errors.As(err, &exit) {
		code = ErrCode(exit.ExitCode())
	}
	return code
}

func (c *Cmd) Run() ErrCode {
	var (
		code ErrCode
		exit *exec.ExitError
		err  = c.Cmd.Run()
	)
	if errors.As(err, &exit) {
		code = ErrCode(exit.ExitCode())
	}
	return code
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

// func closeFile(c interface{}) {
// 	if c, ok := c.(io.Closer); ok {
// 		c.Close()
// 	}
// }
