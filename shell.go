package tish

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/midbel/rw"
	"github.com/midbel/shlex"
	"github.com/midbel/tish/internal/parser"
	"github.com/midbel/tish/internal/token"
	"github.com/midbel/tish/internal/words"
	"golang.org/x/sync/errgroup"
)

const shell = "tish"

var (
	ErrExit     = errors.New("exit")
	ErrReadOnly = errors.New("read only")
	ErrEmpty    = errors.New("empty command")
)

type ExitCode int8

const (
	Success ExitCode = iota
	Failure
)

func (e ExitCode) Success() bool {
	return e == Success
}

func (e ExitCode) Failure() bool {
	return !e.Success()
}

func (e ExitCode) Error() string {
	return fmt.Sprintf("%d", e)
}

var specials = map[string]struct{}{
	"HOME":    {},
	"SECONDS": {},
	"PWD":     {},
	"OLDPWD":  {},
	"PID":     {},
	"PPID":    {},
	"RANDOM":  {},
	"SHELL":   {},
	"?":       {},
	"#":       {},
	"0":       {},
	"$":       {},
	"@":       {},
}

type Shell struct {
	locals   Environment
	alias    map[string][]string
	commands map[string]Command
	find     CommandFinder
	echo     bool

	env map[string]string

	Stack
	now  time.Time
	rand *rand.Rand

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	context struct {
		pid  int
		code int
		name string
		args []string
	}

	builtins map[string]Builtin
}

func New(options ...ShellOption) (*Shell, error) {
	s := Shell{
		now:      time.Now(),
		Stack:    DirectoryStack(),
		alias:    make(map[string][]string),
		commands: make(map[string]Command),
		env:      make(map[string]string),
		builtins: builtins,
	}
	s.rand = rand.New(rand.NewSource(s.now.Unix()))
	cwd, _ := os.Getwd()
	s.Stack.Chdir(cwd)
	for i := range options {
		if err := options[i](&s); err != nil {
			return nil, err
		}
	}
	if s.locals == nil {
		s.locals = EmptyEnv()
	}
	return &s, nil
}

func (s *Shell) Exit() {
	os.Exit(s.context.code)
}

func (s *Shell) SetIn(r io.Reader) {
	s.stdin = r
}

func (s *Shell) SetOut(w io.Writer) {
	s.stdout = w
}

func (s *Shell) SetErr(w io.Writer) {
	s.stderr = w
}

// implements CommandFinder.Find
func (s *Shell) Find(ctx context.Context, name string) (Command, error) {
	if s.find != nil {
		c, err := s.find.Find(ctx, name)
		if err == nil {
			return c, nil
		}
	}
	if c, ok := s.commands[name]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("%s: command not found", name)
}

func (s *Shell) SetEcho(echo bool) {
	s.echo = echo
}

// Rxport defines an environment variable from the shell
func (s *Shell) Export(ident, value string) {
	s.env[ident] = value
}

// Unexport removes an environment variable from the shell
func (s *Shell) Unexport(ident string) {
	delete(s.env, ident)
}

// Alias define a shell alias to the Shell
func (s *Shell) Alias(ident, script string) error {
	alias, err := shlex.Split(strings.NewReader(script))
	if err != nil {
		return err
	}
	s.alias[ident] = alias
	return nil
}

// Unalias remove a shell alias from the Shell
func (s *Shell) Unalias(ident string) {
	delete(s.alias, ident)
}

func (s *Shell) Subshell() (*Shell, error) {
	options := []ShellOption{
		WithEnv(s),
		WithCwd(s.Cwd()),
		WithStdout(s.stdout),
		WithStderr(s.stderr),
		WithStdin(s.stdin),
	}
	if s.echo {
		options = append(options, WithEcho())
	}
	sub, err := New(options...)
	if err != nil {
		return nil, err
	}
	for n, str := range s.alias {
		sub.alias[n] = str
	}
	return sub, nil
}

// implements Environment.Resolve
func (s *Shell) Resolve(ident string) ([]string, error) {
	str, err := s.locals.Resolve(ident)
	if err == nil && len(str) > 0 {
		return str, nil
	}
	if v, ok := s.env[ident]; ok {
		return []string{v}, nil
	}
	if str = s.resolveSpecials(ident); len(str) > 0 {
		return str, nil
	}
	return nil, err
}

// implements Environment.Define
func (s *Shell) Define(ident string, values []string) error {
	if _, ok := specials[ident]; ok {
		return ErrReadOnly
	}
	return s.locals.Define(ident, values)
}

// implements Environment.Delete
func (s *Shell) Delete(ident string) error {
	if _, ok := specials[ident]; ok {
		return ErrReadOnly
	}
	return s.locals.Delete(ident)
}

func (s *Shell) Expand(str string, args []string) ([]string, error) {
	env := getEnvShell(s)
	return parser.Expand(str, args, env)
}

func (s *Shell) Dry(str, cmd string, args []string) error {
	s.setContext(cmd, args)
	defer s.clearContext()

	return parser.ExpandWith(str, args, s, func(str [][]string) {
		for i := range str {
			io.WriteString(s.stdout, strings.Join(str[i], " "))
			io.WriteString(s.stdout, "\n")
		}
	})
}

func (s *Shell) Register(list ...Command) {
	for i := range list {
		s.commands[list[i].Command()] = list[i]
	}
}

func (s *Shell) Run(ctx context.Context, r io.Reader, cmd string, args []string) error {
	s.setContext(cmd, args)
	defer s.clearContext()
	var (
		p   = parser.NewParser(r)
		ret error
	)
	for {
		ex, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return ret
			}
			return err
		}
		ret = s.execute(ctx, ex)
	}
}

func (s *Shell) Execute(ctx context.Context, str, cmd string, args []string) error {
	return s.Run(ctx, strings.NewReader(str), cmd, args)
}

func (s *Shell) execute(ctx context.Context, ex words.Executer) error {
	var err error
	switch ex := ex.(type) {
	case words.ExecSimple:
		err = s.executeSingle(ctx, ex.Expander, ex.Redirect)
	case words.ExecList:
		for i := range ex {
			if err = s.execute(ctx, ex[i]); err != nil {
				break
			}
		}
	case words.ExecSubshell:
		return s.executeSubshell(ctx, ex)
	case words.ExecAssign:
		err = s.executeAssign(ex)
	case words.ExecAnd:
		if err = s.execute(ctx, ex.Left); err != nil || s.context.code != 0 {
			break
		}
		err = s.execute(ctx, ex.Right)
	case words.ExecOr:
		if err = s.execute(ctx, ex.Left); err == nil || s.context.code == 0 {
			break
		}
		err = s.execute(ctx, ex.Right)
	case words.ExecPipe:
		err = s.executePipe(ctx, ex)
	case words.ExecFor:
		err = s.executeFor(ctx, ex)
	case words.ExecWhile:
		err = s.executeWhile(ctx, ex)
	case words.ExecUntil:
		err = s.executeUntil(ctx, ex)
	case words.ExecIf:
		err = s.executeIf(ctx, ex)
	case words.ExecCase:
		err = s.executeCase(ctx, ex)
	case words.ExecBreak:
		err = words.ErrBreak
	case words.ExecContinue:
		err = words.ErrContinue
	case words.ExecTest:
		err = s.executeTest(ctx, ex)
	default:
		err = fmt.Errorf("unsupported executer type %T", ex)
	}
	return err
}

func (s *Shell) executeSubshell(ctx context.Context, ex words.ExecSubshell) error {
	sh, err := s.Subshell()
	if err != nil {
		return err
	}
	for i := range ex {
		if err := sh.execute(ctx, ex[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Shell) executeCase(ctx context.Context, ex words.ExecCase) error {
	var (
		env       = getEnvShell(s)
		word, err = ex.Word.Expand(env, false)
	)
	if err != nil {
		return err
	}
	if len(word) != 1 {
		return fmt.Errorf("too many word expanded %s", word)
	}
	for _, i := range ex.List {
		var found bool
		for j := range i.List {
			vs, err := i.List[j].Expand(env, false)
			if err != nil {
				return err
			}
			if len(vs) != 1 {
				return fmt.Errorf("too many word expanded %s", vs)
			}
			if found = vs[0] == word[0]; found {
				break
			}
		}
		if found {
			return s.execute(ctx, i.Body)
		}
	}
	if ex.Default != nil {
		return s.execute(ctx, ex.Default)
	}
	return nil
}

func (s *Shell) executeTest(_ context.Context, ex words.ExecTest) error {
	ok, err := ex.Test(s)
	if err != nil || !ok {
		s.context.code = 1
	} else {
		s.context.code = 0
	}
	return err
}

func (s *Shell) executeFor(ctx context.Context, ex words.ExecFor) error {
	var (
		env       = getEnvShell(s)
		list, err = ex.Expand(env, false)
	)
	if err != nil || len(list) == 0 {
		return s.execute(ctx, ex.Alt)
	}
	for i := range list {
		if err := s.Define(ex.Ident, []string{list[i]}); err != nil {
			return err
		}
		if err := s.execute(ctx, ex.Body); err != nil {
			if errors.Is(err, words.ErrBreak) {
				break
			}
			if errors.Is(err, words.ErrContinue) {
				continue
			}
			return err
		}
	}
	return nil
}

func (s *Shell) executeWhile(ctx context.Context, ex words.ExecWhile) error {
	var it int
	for {
		err := s.execute(ctx, ex.Cond)
		if err != nil {
			return err
		}
		if s.context.code != 0 {
			break
		}
		it++
		err = s.execute(ctx, ex.Body)
		if err != nil {
			if errors.Is(err, words.ErrBreak) {
				break
			}
			if errors.Is(err, words.ErrContinue) {
				continue
			}
			return err
		}
	}
	if it == 0 {
		return s.execute(ctx, ex.Alt)
	}
	return nil
}

func (s *Shell) executeUntil(ctx context.Context, ex words.ExecUntil) error {
	var it int
	for {
		err := s.execute(ctx, ex.Cond)
		if err != nil {
			return err
		}
		if s.context.code == 0 {
			break
		}
		it++
		err = s.execute(ctx, ex.Body)
		if err != nil {
			if errors.Is(err, words.ErrBreak) {
				break
			}
			if errors.Is(err, words.ErrContinue) {
				continue
			}
			return err
		}
	}
	if it == 0 {
		return s.execute(ctx, ex.Alt)
	}
	return nil
}

func (s *Shell) executeIf(ctx context.Context, ex words.ExecIf) error {
	err := s.execute(ctx, ex.Cond)
	if err != nil {
		return err
	}
	if s.context.code == 0 {
		return s.execute(ctx, ex.Csq)
	}
	return s.execute(ctx, ex.Alt)
}

func (s *Shell) executeSingle(ctx context.Context, ex words.Expander, redirect []words.ExpandRedirect) error {
	str, err := s.expand(ex)
	if err != nil {
		return err
	}
	s.trace(str)
	cmd := s.resolveCommand(ctx, str)

	rd, err := s.setupRedirect(redirect, false)
	if err != nil {
		return err
	}
	defer rd.Close()

	cmd.SetOut(rd.out)
	cmd.SetErr(rd.err)
	cmd.SetIn(rd.in)

	err = cmd.Run()
	s.updateContext(cmd)
	return err
}

func (s *Shell) executePipe(ctx context.Context, ex words.ExecPipe) error {
	var cs []Command
	for i := range ex.List {
		sex, ok := ex.List[i].Executer.(words.ExecSimple)
		if !ok {
			return fmt.Errorf("single command expected")
		}
		str, err := s.expand(sex.Expander)
		if err != nil {
			return err
		}
		cmd := s.resolveCommand(ctx, str)
		rd, err := s.setupRedirect(sex.Redirect, true)
		if err != nil {
			return err
		}
		defer rd.Close()

		cmd.SetIn(rd.in)
		cmd.SetOut(rd.out)
		cmd.SetErr(rd.err)

		cs = append(cs, cmd)
	}
	var (
		err error
		grp errgroup.Group
	)
	for i := 0; i < len(cs)-1; i++ {
		var (
			curr = cs[i]
			next = cs[i+1]
			in   io.ReadCloser
		)
		if in, err = curr.StdoutPipe(); err != nil {
			return err
		}
		next.SetIn(in)
		grp.Go(curr.Start)
	}
	cmd := cs[len(cs)-1]
	cmd.SetOut(s.stdout)
	cmd.SetErr(s.stderr)
	grp.Go(func() error {
		err := cmd.Run()
		s.updateContext(cmd)
		return err
	})
	return grp.Wait()
}

func (s *Shell) executeAssign(ex words.ExecAssign) error {
	var (
		env      = getEnvShell(s)
		str, err = ex.Expand(env, true)
	)
	if err != nil {
		return err
	}
	return s.Define(ex.Ident, str)
}

func (s *Shell) expand(ex words.Expander) ([]string, error) {
	var (
		env      = getEnvShell(s)
		str, err = ex.Expand(env, true)
	)
	if err != nil {
		return nil, err
	}
	if len(str) == 0 {
		return nil, ErrEmpty
	}
	alias, ok := s.alias[str[0]]
	if ok {
		as := make([]string, len(alias))
		copy(as, alias)
		str = append(as, str[1:]...)
	}
	return str, nil
}

func (s *Shell) setContext(name string, args []string) {
	s.context.name = name
	s.context.args = append(s.context.args[:0], args...)
}

func (s *Shell) updateContext(cmd Command) {
	var pid, code int
	if cmd == nil {
		code = 255
	} else {
		pid, code = cmd.Exit()
	}
	s.context.pid = pid
	s.context.code = code
}

func (s *Shell) clearContext() {
	s.context.name = ""
	s.context.args = nil
}

func (s *Shell) resolveCommand(ctx context.Context, str []string) Command {
	if b, ok := s.builtins[str[0]]; ok && b.IsEnabled() {
		b.shell = s
		b.args = str[1:]
		return &b
	}
	var cmd Command
	if c, err := s.Find(ctx, str[0]); err == nil {
		cmd = c
	} else {
		cmd = StandardContext(ctx, str[0], s.Cwd(), str[1:])
	}
	if a, ok := cmd.(interface{ SetArgs([]string) }); ok {
		a.SetArgs(str[1:])
	}
	if e, ok := cmd.(interface{ SetEnv([]string) }); ok {
		e.SetEnv(s.environ())
	}
	return cmd
}

func (s *Shell) resolveSpecials(ident string) []string {
	var ret []string
	switch ident {
	case "SHELL":
		ret = append(ret, shell)
	case "HOME":
		u, err := user.Current()
		if err == nil {
			ret = append(ret, u.HomeDir)
		}
	case "SECONDS":
		sec := time.Since(s.now).Seconds()
		ret = append(ret, strconv.FormatInt(int64(sec), 10))
	case "PWD":
		ret = append(ret, s.Cwd())
	case "OLDPWD":
		// ret = append(ret, s.old)
	case "PID", "$":
		str := strconv.Itoa(os.Getpid())
		ret = append(ret, str)
	case "PPID":
		str := strconv.Itoa(os.Getppid())
		ret = append(ret, str)
	case "RANDOM":
		str := strconv.Itoa(s.rand.Int())
		ret = append(ret, str)
	case "0":
		ret = append(ret, s.context.name)
	case "#":
		ret = append(ret, strconv.Itoa(len(s.context.args)))
	case "?":
		ret = append(ret, strconv.Itoa(s.context.code))
	case "!":
		ret = append(ret, strconv.Itoa(s.context.pid))
	case "*":
		ret = append(ret, strings.Join(s.context.args, " "))
	case "@":
		ret = s.context.args
	default:
		n, err := strconv.Atoi(ident)
		if err != nil {
			break
		}
		var arg string
		if n >= 1 && n <= len(s.context.args) {
			arg = s.context.args[n-1]
		}
		ret = append(ret, arg)
	}
	return ret
}

func (s *Shell) trace(str []string) {
	if !s.echo {
		return
	}
	fmt.Fprintln(s.stdout, strings.Join(str, " "))
}

func (s *Shell) environ() []string {
	var str []string
	for n, v := range s.env {
		str = append(str, fmt.Sprintf("%s=%s", n, v))
	}
	return str
}

const (
	flagRead   = os.O_CREATE | os.O_RDONLY
	flagWrite  = os.O_CREATE | os.O_WRONLY
	flagAppend = os.O_CREATE | os.O_WRONLY | os.O_APPEND
)

func replaceFile(file string, flag int, list ...*os.File) (*os.File, error) {
	fd, err := os.OpenFile(file, flag, 0644)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i] == nil {
			continue
		}
		list[i].Close()
	}
	return fd, nil
}

type redirect struct {
	in  io.ReadCloser
	out io.WriteCloser
	err io.WriteCloser
}

func (r redirect) Close() error {
	for _, c := range []io.Closer{r.in, r.out, r.err} {
		if c == nil {
			continue
		}
		c.Close()
	}
	return nil
}

func (s *Shell) setupRedirect(rs []words.ExpandRedirect, pipe bool) (redirect, error) {
	var (
		stdin  *os.File
		stdout *os.File
		stderr *os.File
		rd     redirect
		env    = getEnvShell(s)
	)
	for _, r := range rs {
		str, err := r.Expand(env, true)
		if err != nil {
			return rd, err
		}
		switch file := str[0]; r.Type {
		case token.RedirectIn:
			stdin, err = replaceFile(file, flagRead, stdin)
		case token.RedirectOut:
			if stdout == stderr {
				stdout = nil
			}
			stdout, err = replaceFile(file, flagWrite, stdout)
		case token.RedirectErr:
			if stderr == stdout {
				stderr = nil
			}
			stderr, err = replaceFile(file, flagWrite, stderr)
		case token.RedirectBoth:
			var fd *os.File
			if fd, err = replaceFile(file, flagWrite, stdout, stderr); err == nil {
				stdout, stderr = fd, fd
			}
		case token.AppendOut:
			if stdout.Fd() == stderr.Fd() {
				stdout = nil
			}
			stdout, err = replaceFile(file, flagAppend, stdout)
		case token.AppendErr:
			if stderr == stdout {
				stderr = nil
			}
			stderr, err = replaceFile(file, flagAppend, stderr)
		case token.AppendBoth:
			var fd *os.File
			if fd, err = replaceFile(file, flagAppend, stdout, stderr); err == nil {
				stdout, stderr = fd, fd
			}
		default:
			err = fmt.Errorf("unknown/unsupported redirection")
		}
		if err != nil {
			return rd, err
		}
	}
	rd.in = fileOrReader(stdin, s.stdin, pipe)
	rd.out = fileOrWriter(stdout, s.stdout, pipe)
	rd.err = fileOrWriter(stderr, s.stderr, pipe)
	return rd, nil
}

func fileOrWriter(f *os.File, w io.Writer, pipe bool) io.WriteCloser {
	if f == nil {
		if pipe {
			return nil
		}
		return rw.NopWriteCloser(w)
	}
	return f
}

func fileOrReader(f *os.File, r io.Reader, pipe bool) io.ReadCloser {
	if f == nil {
		if pipe {
			return nil
		}
		if r == nil {
			r = rw.Empty()
		}
		return rw.NopReadCloser(r)
	}
	return f
}
