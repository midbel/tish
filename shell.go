package tish

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

const (
	ExitOk int = iota
	ExitKo
	ExitHelp
	ExitUsage
	ExitExec
	ExitNotExec
)

type Process interface {
	Run() error
	Start() error
	Wait() error
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

	env  *Env
	vars *Env

	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader

	proc struct {
		exit int
		pid  int
		cmd  string
		args []string
		sys  time.Duration
		user time.Duration
	}

	alias map[string]string
}

func NewShell(r io.Reader, options ...Option) (*Shell, error) {
	psr, err := NewParser(r)
	if err != nil {
		return nil, err
	}
	s := Shell{
		psr:   psr,
		env:   EmptyEnv(),
		vars:  EmptyEnv(),
		now:   time.Now(),
		alias: make(map[string]string),
	}
	for _, o := range options {
		if err := o(&s); err != nil {
			return nil, err
		}
	}

	return &s, nil
}

func (s *Shell) Uptime() time.Duration {
	return time.Since(s.now)
}

func (s *Shell) Execute() (int, error) {
	var (
		cmd Command
		err error
	)
	for err == nil {
		cmd, err = s.psr.Parse()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return ExitKo, err
		}
		err = s.execute(cmd)
	}
	return s.proc.exit, err
}

func (s *Shell) execute(cmd Command) error {
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
	case Continue:
	default:
		return fmt.Errorf("unsupported command type %T", cmd)
	}
	return nil
}

func (s *Shell) executeFor(cmd For) {
	s.vars = EnclosedEnv(s.vars)
	for _, w := range cmd.words {
		s.vars.Define(cmd.ident.Literal, w)
		s.execute(cmd.body)
	}
	s.vars = s.vars.Unwrap()
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
	var (
		env = MergeEnv(s.vars, s.env)
		str = cmd.word.Expand(env)
	)
	for _, c := range cmd.clauses {
		if c.Match(str, env) {
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
	executeAssignWithEnv(cmd, s.vars)
}

func (s *Shell) executeSimple(cmd Simple) {
	s.env = EnclosedEnv(s.env)
	for _, a := range cmd.env {
		executeAssignWithEnv(a, s.env)
	}

	ident, args := s.prepare(cmd.words)
	s.run(ident, args)
	s.env = s.env.Unwrap()
}

func (s *Shell) run(ident string, args []string) {
	if ident == "" {
		return
	}
	var exe Process
	if b, ok := builtins[ident]; ok && b.Runnable() {
		b.Args = args
		b.Shell = s

		exe = &b
	} else {
		cmd := exec.Command(ident, args...)
		cmd.Env = s.env.Environ()
		exe = wrapCmd(cmd)
	}

	s.attachIn(exe)
	s.attachOut(exe)
	s.attachErr(exe)
	defer exe.Close()

	exe.Run()

	s.proc.cmd = ident
	s.proc.args = args
	switch exe := exe.(type) {
	case *Cmd:
		s.proc.exit = exe.Cmd.ProcessState.ExitCode()
		s.proc.pid = exe.Cmd.ProcessState.Pid()
		s.proc.sys = exe.Cmd.ProcessState.SystemTime()
		s.proc.user = exe.Cmd.ProcessState.UserTime()
	case *Builtin:
		s.proc.exit = exe.Exit
		s.proc.pid = s.pid
	}
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

func (s *Shell) prepare(words []Word) (string, []string) {
	var ws []string
	for _, w := range words {
		ws = append(ws, w.Expand(s.env))
	}
	if len(ws) == 0 {
		return "", nil
	}
	name := ws[0]
	if n, ok := s.alias[name]; ok {
		name = n
	}
	if len(ws) > 1 {
		return name, ws[1:]
	}
	return name, nil
}

func executeAssignWithEnv(cmd Assign, env *Env) {
	env.Define(cmd.ident.Literal, cmd.word)
}

type Cmd struct {
	*exec.Cmd
}

func wrapCmd(c *exec.Cmd) Process {
	cmd := Cmd{Cmd: c}
	return &cmd
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
