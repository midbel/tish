package tish

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

type Status struct {
	Exit int
	Pid  int
	Err  error
}

var (
	ErrExit     = errors.New("exit")
	ErrBreak    = errors.New("break")
	ErrContinue = errors.New("continue")
)

const (
	ExitOk int = iota
	ExitKo
	ExitHelp
	ExitUsage
	ExitExec
	ExitNotExec

	ExitQuit = 255
)

type Process interface {
	Execute() Status
	Close() error
}

type Option func(*Shell) error

func WithArgs(args []string) Option {
	return func(s *Shell) error {
		if len(args) > 0 {
			s.args = append(s.args[:0], args...)
		}
		return nil
	}
}

func WithStdin(r io.Reader) Option {
	return func(s *Shell) error {
		s.stdin = r
		return nil
	}
}

func WithStdout(w io.Writer) Option {
	return func(s *Shell) error {
		s.stdout = w
		return nil
	}
}

func WithStderr(w io.Writer) Option {
	return func(s *Shell) error {
		s.stderr = w
		return nil
	}
}

type Shell struct {
	psr *Parser

	args  []string
	depth int

	now time.Time
	pid int
	uid int

	env  Environment
	vars Environment

	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader

	proc struct {
		exit int
		pid  int
		cmd  string
		args []string
	}

	alias map[string][]string
}

func NewShell(r io.Reader, options ...Option) (*Shell, error) {
	psr, err := NewParser(r)
	if err != nil {
		return nil, err
	}
	var (
		env  = EmptyEnv()
		vars = EnclosedEnv(env)
	)
	s := Shell{
		psr:   psr,
		env:   env,
		vars:  vars,
		now:   time.Now(),
		uid:   os.Getuid(),
		pid:   os.Getpid(),
		alias: make(map[string][]string),
	}
	for _, o := range options {
		if err := o(&s); err != nil {
			return nil, err
		}
	}
	return &s, nil
}

func (s *Shell) RegisterAlias(ident string, words []string) {
	s.alias[ident] = words
}

func (s *Shell) UnregisterAlias(is ...string) {
	if len(is) == 0 {
		for k := range s.alias {
			is = append(is, k)
		}
	}
	for _, i := range is {
		delete(s.alias, i)
	}
}

func (s *Shell) Define(id, value string) error {
	return s.vars.Define(id, value)
}

func (s *Shell) Resolve(id string) string {
	return s.vars.Resolve(id)
}

func (s *Shell) Delete(id string) error {
	return s.vars.Delete(id)
}

func (s *Shell) Environ() []string {
	return s.vars.Environ()
}

func (s *Shell) Execute() (int, error) {
	var (
		cmd Command
		err error
	)
	for err == nil {
		cmd, err = s.psr.Parse()
		if errors.Is(err, io.EOF) {
			err = nil
			break
		}
		if err != nil {
			return ExitKo, err
		}
		err = s.execute(cmd)
	}
	if s.proc.exit >= ExitQuit {
		s.proc.exit -= ExitQuit
	}
	if errors.Is(err, ErrExit) {
		err = nil
	}
	return s.proc.exit, err
}

func (s *Shell) execute(cmd Command) error {
	if s.proc.exit >= ExitQuit {
		return ErrExit
	}
	if cmd == nil {
		return nil
	}
	switch cmd := cmd.(type) {
	case List:
		s.executeList(cmd)
	case Simple:
		s.executeSimple(cmd)
	case And:
		s.executeAnd(cmd)
	case Or:
		s.executeOr(cmd)
	case Assign:
		s.executeAssign(cmd)
	case For:
		s.executeFor(cmd)
	case Until:
		s.executeUntil(cmd)
	case While:
		s.executeWhile(cmd)
	case Case:
		s.executeCase(cmd)
	case If:
		s.executeIf(cmd)
	case Break:
		return ErrBreak
	case Continue:
		return ErrContinue
	default:
		return fmt.Errorf("unsupported command type %T", cmd)
	}
	return nil
}

func (s *Shell) executeFor(cmd For) {
}

func (s *Shell) executeWhile(cmd While) {
	for {
		s.execute(cmd.cmd)
		if s.proc.exit != 0 {
			break
		}
		s.execute(cmd.body)
	}
}

func (s *Shell) executeUntil(cmd Until) {
	for {
		s.execute(cmd.cmd)
		if s.proc.exit == 0 {
			break
		}
		s.execute(cmd.body)
	}
}

func (s *Shell) executeIf(cmd If) {
	var next Command

	s.execute(cmd.cmd)
	if s.proc.exit == 0 {
		next = cmd.csq
	} else {
		next = cmd.alt
	}
	s.execute(next)
}

func (s *Shell) executeCase(cmd Case) {
	str := cmd.word.Expand(s.vars)
	for _, c := range cmd.clauses {
		if c.Match(str, s.vars) {
			s.execute(c.body)
			break
		}
	}
}

func (s *Shell) executeList(cmd List) {
	for i := range cmd.cmds {
		s.execute(cmd.cmds[i])
	}
}

func (s *Shell) executeAnd(cmd And) {
	s.execute(cmd.left)
	if s.proc.exit != 0 {
		return
	}
	s.execute(cmd.right)
}

func (s *Shell) executeOr(cmd Or) {
	s.execute(cmd.left)
	if s.proc.exit == 0 {
		return
	}
	s.execute(cmd.right)
}

func (s *Shell) executeAssign(cmd Assign) {
	
}

func (s *Shell) executeSimple(cmd Simple) {

}

func (s *Shell) attachIn(exe Process) error {
	in, err := NewReader(s.stdin)
	if err != nil {
		return err
	}
	switch e := exe.(type) {
	case *Cmd:
		e.Cmd.Stdin = in
	case *Builtin:
		e.Stdin = in
	default:
		err = fmt.Errorf("unsupported process type %T", e)
	}
	return err
}

func (s *Shell) attachOut(exe Process) error {
	out, err := NewWriter(s.stdout)
	if err != nil {
		return err
	}
	switch e := exe.(type) {
	case *Cmd:
		e.Cmd.Stdout = out
	case *Builtin:
		e.Stdout = out
	default:
		err = fmt.Errorf("unsupported process type %T", e)
	}
	return err
}

func (s *Shell) attachErr(exe Process) error {
	out, err := NewWriter(s.stderr)
	if err != nil {
		return err
	}
	switch e := exe.(type) {
	case *Cmd:
		e.Cmd.Stderr = out
	case *Builtin:
		e.Stderr = out
	default:
		err = fmt.Errorf("unsupported process type %T", e)
	}
	return err
}

type Cmd struct {
	*exec.Cmd
}

func wrapCmd(c *exec.Cmd) Process {
	cmd := Cmd{Cmd: c}
	return &cmd
}

func (c *Cmd) Execute() Status {
	err := c.Cmd.Run()
	return Status{
		Exit: c.ProcessState.ExitCode(),
		Pid:  c.ProcessState.Pid(),
		Err:  err,
	}
}

func (c *Cmd) Close() error {
	if c, ok := c.Cmd.Stdin.(io.Closer); ok {
		c.Close()
	}
	if c, ok := c.Cmd.Stdout.(io.Closer); ok {
		c.Close()
	}
	if c, ok := c.Cmd.Stderr.(io.Closer); ok {
		c.Close()
	}
	return nil
}

type reader struct {
	inner  io.ReadCloser
	writer io.Closer
}

func NewReader(r io.Reader) (io.ReadCloser, error) {
	rs, ws, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	i := reader{
		inner:  rs,
		writer: ws,
	}
	go io.Copy(ws, r)
	return &i, nil
}

func (r *reader) Read(bs []byte) (int, error) {
	return r.inner.Read(bs)
}

func (r *reader) Close() error {
	r.inner.Close()
	return r.writer.Close()
}

type writer struct {
	inner  io.WriteCloser
	reader io.Closer
}

func NewWriter(w io.Writer) (io.WriteCloser, error) {
	rs, ws, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	i := writer{
		inner:  ws,
		reader: rs,
	}
	go io.Copy(w, rs)
	return &i, nil
}

func (w *writer) Write(bs []byte) (int, error) {
	return w.inner.Write(bs)
}

func (w *writer) Close() error {
	w.inner.Close()
	return w.reader.Close()
}
