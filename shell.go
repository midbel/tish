package tish

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var ErrFailed = errors.New("process terminated with failure")

const MaxHistSize = 100

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
	var (
		in  = os.NewFile(os.Stdin.Fd(), "stdin")
		out = os.NewFile(os.Stdout.Fd(), "stdout")
		err = os.NewFile(os.Stderr.Fd(), "stderr")
	)
	return NewShell(in, out, err)
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

func (s *Shell) Execute() {
	w, err := s.parseArgs()
	if err != nil {
		fmt.Fprintln(s.stderr, err)
		s.Exit(1)
	}
	if err := s.execute(w); err != nil {
		fmt.Fprintln(s.stderr, err)
		s.Exit(2)
	}
	s.Exit(s.proc.exit)
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
	s.proc.pid = 0
	s.proc.exit = ExitOk

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
	return nil
}

func (s *Shell) executeSimple(ws []Word) error {
	var (
		args = make([]string, 0, len(ws)*4)
		env  = NewEnvironment()
		rs   []Redirect
	)
	for _, w := range ws {
		if a, ok := w.(Assignment); ok {
			xs, err := a.Expand(s.locals)
			if err != nil {
				return err
			}
			env.Set(a.ident, xs)
			continue
		}
		if r, ok := w.(Redirect); ok {
			rs = append(rs, r)
		}
		xs, err := w.Expand(s.locals)
		if err != nil {
			return err
		}
		args = append(args, xs...)
	}

	cmd, err := s.prepare(args, env.Environ())
	if err != nil {
		return err
	}
	for _, r := range rs {
		f, err := r.Open(s.locals)
		if err != nil {
			return err
		}
		if err := cmd.Replace(r.file, f); err != nil {
			return err
		}
	}
	s.proc.exit = cmd.Run()
	if p, ok := cmd.(interface{ Pid() int }); ok {
		s.proc.pid = p.Pid()
	}
	return nil
}

func (s *Shell) executeLiteral(i Literal) error {
	args, err := i.Expand(s.locals)
	if err != nil {
		return err
	}
	if s.Verbose {
		fmt.Fprintln(s.stdout, strings.Join(args, " "))
	}
	if s.Dry {
		return nil
	}
	if w, ok := s.alias[args[0]]; ok {
		vs, err := w.Expand(s.locals)
		if err != nil {
			return err
		}
		vs = append(vs, args[1:]...)
	}
	cmd, err := s.prepare(args, nil)
	if err != nil {
		return err
	}
	s.proc.exit = cmd.Run()
	if p, ok := cmd.(interface{ Pid() int }); ok {
		s.proc.pid = p.Pid()
	}
	return nil
}

func (s *Shell) prepare(args []string, env []string) (Command, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command given")
	}
	s.proc.pid = 0
	s.proc.exit = ExitOk

	in := s.duplicateReader(s.stdin)
	out := s.duplicateWriter(s.stdout)
	err := s.duplicateWriter(s.stderr)

	if c, ok := builtins[args[0]]; ok && c.Runnable() {
		c.stdin = in
		c.stdout = out
		c.stderr = err

		c.Shell = s
		c.Args = args[1:]

		return &c, nil
	}
	c := exec.Command(args[0], args[1:]...)

	if es := s.globals.Environ(); len(es) > 0 {
		c.Env = append(c.Env, es...)
	}
	if len(env) > 0 {
		c.Env = append(c.Env, env...)
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
	case "PWD":
		vs = append(vs, s.Cwd())
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

func (s *Shell) parseArgs() (Word, error) {
	flag.BoolVar(&s.Verbose, "v", false, "print commands that will be executed on stderr")
	flag.BoolVar(&s.Dry, "n", false, "dry run")
	cmd := flag.Bool("c", false, "read command from the command string")
	flag.Parse()

	var r io.Reader
	if *cmd {
		r = strings.NewReader(flag.Arg(0))
	} else {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}
	for i := 1; i < flag.NArg(); i++ {
		s.Args = append(s.Args, flag.Arg(i))
	}
	return Parse(r)
}

type Cmd struct {
	*exec.Cmd
}

func (c *Cmd) Replace(fd int, f *os.File) error {
	switch fd {
	case fdIn:
		closeFile(c.Stdin)
		c.Stdin = f
	case fdOut:
		closeFile(c.Stdout)
		c.Stdout = f
	case fdErr:
		c.Stderr = f
		closeFile(c.Stderr)
	case fdBoth:
		closeFile(c.Stdout)
		closeFile(c.Stderr)
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

func closeFile(c interface{}) {
	if c, ok := c.(io.Closer); ok {
		c.Close()
	}
}
