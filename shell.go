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
	"github.com/midbel/tish/internal/stdio"
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

type Shell struct {
	locals   Environment
	alias    map[string][]string
	commands map[string]Command
	find     CommandFinder
	depth    int
	echo     bool

	env map[string]string

	Stack
	History
	now  time.Time
	rand *rand.Rand

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	context struct {
		// PID of last executed command
		pid int
		// exit code of last executed command
		code int
		// name of last executed command
		name string
		// arguments of last executed command
		args []string
	}

	builtins map[string]Builtin
}

func New(options ...ShellOption) (*Shell, error) {
	sh := Shell{
		now:      time.Now(),
		Stack:    DirectoryStack(),
		History:  HistoryStack(),
		alias:    make(map[string][]string),
		commands: make(map[string]Command),
		env:      make(map[string]string),
		builtins: builtins,
	}
	sh.rand = rand.New(rand.NewSource(sh.now.Unix()))
	cwd, _ := os.Getwd()
	sh.Stack.Chdir(cwd)
	for i := range options {
		if err := options[i](&sh); err != nil {
			return nil, err
		}
	}
	if sh.stdin == nil {
		sh.stdin = rw.Empty()
	}
	if sh.locals == nil {
		sh.locals = EmptyEnv()
	}
	return &sh, nil
}

func (s *Shell) Close() error {
	for _, c := range []interface{}{s.stdin, s.stdout, s.stderr} {
		if c == nil {
			continue
		}
		if c, ok := c.(io.Closer); ok {
			c.Close()
		}
	}
	return nil
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
	sub.depth = s.depth + 1
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

	env := getEnvShell(s)
	return parser.ExpandWith(str, args, env, func(str [][]string) {
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
	ok, err := ex.Test(getEnvShell(s))
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
	cmd.SetOut(s.stdout)
	cmd.SetErr(s.stderr)
	cmd.SetIn(s.stdin)

	// rd, err := s.setupRedirect(redirect, false)
	// if err != nil {
	// 	return err
	// }
	// defer rd.Close()
	//
	// cmd.SetOut(rd.out)
	// cmd.SetErr(rd.err)
	// cmd.SetIn(rd.in)

	err = cmd.Run()
	s.updateContext(cmd)
	return err
}

func (s *Shell) executePipe(ctx context.Context, ex words.ExecPipe) error {
	get := func(ex words.Executer) (Command, error) {
		sex, ok := ex.(words.ExecSimple)
		if !ok {
			return nil, fmt.Errorf("single command expected")
		}
		str, err := s.expand(sex.Expander)
		if err != nil {
			return nil, err
		}
		cmd := s.resolveCommand(ctx, str)
		cmd.SetErr(s.stderr)
		cmd.SetIn(s.stdin)
		return cmd, nil
	}
	run := func(cmd Command, update bool) func() error {
		return func() error {
			err := cmd.Run()
			if update {
				s.updateContext(cmd)
			}
			return err
		}
	}
	var (
		last = len(ex.List) - 1
		in   io.ReadCloser
		grp  errgroup.Group
	)
	cmd, err := get(ex.List[last].Executer)
	if err != nil {
		return err
	}
	cmd.SetOut(s.stdout)
	cmd.SetErr(s.stderr)
	for i := last - 1; i >= 0; i-- {
		curr, err := get(ex.List[i].Executer)
		if err != nil {
			return err
		}
		if in, err = curr.StdoutPipe(); err != nil {
			return err
		}
		defer in.Close()

		cmd.SetIn(in)
		grp.Go(run(cmd, i == last-1))
		cmd = curr
	}
	cmd.SetIn(s.stdin)
	err = cmd.Run()
	if err1 := grp.Wait(); err1 != nil && err == nil {
		err = err1
	}
	return err
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
		b.Args = str[1:]
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
	case varShell:
		ret = append(ret, shell)
	case varSub:
		ret = append(ret, strconv.Itoa(s.depth))
	case varHome:
		u, err := user.Current()
		if err == nil {
			ret = append(ret, u.HomeDir)
		}
	case varSeconds:
		sec := time.Since(s.now).Seconds()
		ret = append(ret, strconv.FormatInt(int64(sec), 10))
	case varPwd:
		ret = append(ret, s.Cwd())
	case varOld:
		// ret = append(ret, s.old)
	case varPid, varShellPid:
		str := strconv.Itoa(os.Getpid())
		ret = append(ret, str)
	case varPpid:
		str := strconv.Itoa(os.Getppid())
		ret = append(ret, str)
	case varRand:
		str := strconv.Itoa(s.rand.Int())
		ret = append(ret, str)
	case varScript:
		ret = append(ret, s.context.name)
	case varNarg:
		ret = append(ret, strconv.Itoa(len(s.context.args)))
	case varExit:
		ret = append(ret, strconv.Itoa(s.context.code))
	case varLastPid:
		ret = append(ret, strconv.Itoa(s.context.pid))
	case varArgsStr:
		ret = append(ret, strings.Join(s.context.args, " "))
	case varArgsArr:
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

func fileOrWriter(f *os.File, w io.Writer, pipe bool) io.Writer {
	if f == nil {
		if pipe {
			return nil // w
		}
		return stdio.Writer(w)
	}
	return stdio.Writer(w)
}

func fileOrReader(f *os.File, r io.Reader, pipe bool) io.ReadCloser {
	if f == nil {
		if pipe {
			return nil // stdio.Reader(r)
		}
		if r == nil {
			r = rw.Empty()
		}
		return stdio.Reader(r)
	}
	return stdio.Reader(f)
}

type redirect struct {
	in  io.Reader
	out io.Writer
	err io.Writer
}

func (r redirect) Close() error {
	for _, c := range []interface{}{r.in, r.out, r.err} {
		if c == nil {
			continue
		}
		if c, ok := c.(io.Closer); ok {
			c.Close()
		}
	}
	return nil
}
