package tish

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"time"
)

const (
	tishName    = "tish"
	tishVersion = "0.0.1"
	maxSubshell = 255
)

type Shell struct {
	parser *Parser
	locals Environment
	env    Environment

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	builtins map[string]builtin

	level int
	when  time.Time
	dirs  []string
	path  []string
	exext []string

	exec struct {
		code int
		cmd  string
		args []string
	}
}

func NewShell(r io.Reader) (*Shell, error) {
	return NewShellWithEnv(r, EmptyEnv())
}

func NewShellWithEnv(r io.Reader, locals Environment) (*Shell, error) {
	p, err := New(r)
	if err != nil {
		return nil, err
	}
	if locals == nil {
		locals = EmptyEnv()
	}
	s := Shell{
		parser:   p,
		env:      EmptyEnv(),
		locals:   locals,
		builtins: builtins,
		level:    1,
		Stdout:   NopCloser(os.Stdout),
		Stderr:   NopCloser(os.Stderr),
		when:     time.Now(),
		path:     filepath.SplitList(os.Getenv("PATH")),
	}
	if cwd, err := os.Getwd(); err == nil {
		s.dirs = append(s.dirs, cwd)
	}
	rand.Seed(s.when.Unix())
	return &s, nil
}

func (s *Shell) SetDirs(dirs []string) {
	s.path = append(s.path, dirs...)
}

func (s *Shell) SetExts(exts ...string) {
	s.exext = append(s.exext, exts...)
}

func (s *Shell) Sub() (*Shell, error) {
	if s.level == maxSubshell {
		return nil, fmt.Errorf("too many subshell created")
	}
	sub := *s

	sub.level++
	sub.env = EnclosedEnv(s.env)
	sub.locals = EnclosedEnv(s.locals)
	sub.dirs = slices.Clone(s.dirs)
	sub.path = slices.Clone(s.path)
	sub.exec.code = 0
	sub.exec.cmd = ""
	sub.exec.args = []string{}

	return &sub, nil
}

func (s *Shell) Run() error {
	for {
		cmd, err := s.parser.Parse()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if err = s.execute(cmd); err != nil {
			return err
		}
	}
	return nil
}

func (s *Shell) WorkDir() string {
	if n := len(s.dirs); n >= 1 {
		return s.dirs[n-1]
	}
	return "."
}

func (s *Shell) OldWorkDir() string {
	if n := len(s.dirs); n >= 2 {
		return s.dirs[n-2]
	}
	return "."
}

func (s *Shell) execute(cmd Command) error {
	var err error
	switch c := cmd.(type) {
	case cmdSingle:
		err = s.executeSingle(cmd)
	case cmdRedirect:
		err = s.executeRedirect(c)
	case cmdAssign:
		err = s.executeAssign(c)
	case cmdPipe:
		return s.executePipe(c.list)
	case cmdAnd:
		if err = s.execute(c.left); err != nil {
			break
		}
		err = s.execute(c.right)
	case cmdOr:
		if err = s.execute(c.left); err == nil {
			break
		}
		err = s.execute(c.right)
	case cmdFor:
		err = s.executeFor(c)
	case cmdWhile:
		err = s.executeWhile(c)
	case cmdUntil:
		err = s.executeUntil(c)
	case cmdIf:
		err = s.executeIf(c)
	case cmdList:
		err = s.executeList(c)
	case cmdGroup:
		err = s.executeGroup(c)
	default:
		return fmt.Errorf("unsupported command type %T", cmd)
	}
	return s.reset(err)
}

func (s *Shell) reset(err error) error {
	if err == nil {
		s.exec.code = 0
		return err
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		s.exec.code = exit.ExitCode()
	} else {
		s.exec.code = 1
	}
	return err
}

func (s *Shell) executeSingle(cmd Command) error {
	if c, ok := cmd.(cmdSingle); ok && len(c.export) > 0 {
		s.env = EnclosedEnv(s.env)
		if u, ok := s.env.(interface{ unwrap() Environment }); ok {
			defer func() {
				s.env = u.unwrap()
			}()
		}
		for _, e := range c.export {
			a, ok := e.(cmdAssign)
			if !ok {
				return fmt.Errorf("unexpected command type %T", e)
			}
			str, err := a.word.Expand(s)
			if err != nil {
				return err
			}
			s.env.Define(a.ident, str)
		}
	}
	c, err := s.prepare(cmd)
	if err != nil {
		return err
	}
	return c.Run()
}

func (s *Shell) executeRedirect(cmd cmdRedirect) error {
	c, err := s.prepare(cmd.Command)
	if err != nil {
		return err
	}
	var (
		stdin  []io.ReadCloser
		stdout []io.WriteCloser
		stderr []io.WriteCloser
	)
	for _, w := range cmd.redirect {
		r, ok := w.(stdRedirect)
		if !ok {
			return fmt.Errorf("unsupported redirection word")
		}
		switch r.Kind {
		case RedirectIn:
			r, err := openFile(r.Word, s)
			if err != nil {
				return err
			}
			stdin = append(stdin, r)
		case RedirectOut:
			w, err := writeFile(r.Word, s)
			if err != nil {
				return err
			}
			stdout = append(stdout, w)
		case RedirectErr:
			w, err := writeFile(r.Word, s)
			if err != nil {
				return err
			}
			stderr = append(stderr, w)
		case RedirectBoth:
			w, err := writeFile(r.Word, s)
			if err != nil {
				return err
			}
			stderr = append(stderr, w)
			stdout = append(stdout, w)
		case AppendOut:
			w, err := appendFile(r.Word, s)
			if err != nil {
				return err
			}
			stdout = append(stdout, w)
		case AppendErr:
			w, err := appendFile(r.Word, s)
			if err != nil {
				return err
			}
			stderr = append(stderr, w)
		case AppendBoth:
			w, err := appendFile(r.Word, s)
			if err != nil {
				return err
			}
			stderr = append(stderr, w)
			stdout = append(stdout, w)
		case RedirectErrOut:
		case RedirectOutErr:
		default:
			return fmt.Errorf("unsupported redirection type")
		}
	}
	c.replaceIn(multiReader(stdin))
	if len(stdout) > 0 {
		c.replaceOut(multiWriter(stdout))
	}
	if len(stderr) > 0 {
		c.replaceErr(multiWriter(stderr))
	}
	return c.Run()
}

func (s *Shell) executeAssign(cmd cmdAssign) error {
	list, err := cmd.word.Expand(s)
	if err != nil {
		return err
	}
	s.Define(cmd.ident, list)
	return nil
}

func (s *Shell) executePipe(list []Command) error {
	var runPipe func([]Command, io.Reader) error

	runPipe = func(list []Command, in io.Reader) error {
		cmd, err := s.prepare(list[0])
		if err != nil {
			return err
		}
		cmd.replaceIn(in)
		if len(list) == 1 {
			if err := cmd.Start(); err != nil {
				return err
			}
			return cmd.Wait()
		}
		cmd.replaceOut(nil)
		pipe, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return err
		}
		if len(list) > 1 {
			if err := runPipe(list[1:], pipe); err != nil {
				return err
			}
		}
		return cmd.Wait()
	}
	return runPipe(list, s.Stdin)
}

func (s *Shell) executeGroup(cmd cmdGroup) error {
	sub, err := s.Sub()
	if err != nil {
		return err
	}
	for _, c := range cmd.commands {
		if err = sub.execute(c); err != nil {
			break
		}
	}
	return err
}

func (s *Shell) executeList(cmd cmdList) error {
	for i := range cmd.commands {
		if err := s.execute(cmd.commands[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Shell) executeIf(cmd cmdIf) error {
	var ok bool
	if t, test := cmd.test.(Expr); test {
		ok, _ = t.Test(s)
	} else {
		err := s.execute(cmd.test)
		ok = err == nil
	}
	if ok {
		return s.execute(cmd.csq)
	}
	if cmd.alt != nil {
		return s.execute(cmd.alt)
	}
	return nil
}

func (s *Shell) executeUntil(cmd cmdUntil) error {
	s.locals = EnclosedEnv(s.locals)
	if u, ok := s.locals.(interface{ unwrap() Environment }); ok {
		defer func() {
			s.locals = u.unwrap()
		}()
	}
	for {
		if err := s.execute(cmd.iter); err == nil {
			break
		}
		if err := s.execute(cmd.body); err != nil {
			return err
		}
	}
	return nil
}

func (s *Shell) executeWhile(cmd cmdWhile) error {
	s.locals = EnclosedEnv(s.locals)
	if u, ok := s.locals.(interface{ unwrap() Environment }); ok {
		defer func() {
			s.locals = u.unwrap()
		}()
	}
	for {
		if err := s.execute(cmd.iter); err != nil {
			break
		}
		if err := s.execute(cmd.body); err != nil {
			return err
		}
	}
	return nil
}

func (s *Shell) executeFor(cmd cmdFor) error {
	e, err := s.prepare(cmd.iter)
	if err != nil {
		return err
	}
	e.replaceOut(nil)
	r, err := e.StdoutPipe()
	if err != nil {
		return err
	}
	if err := e.Start(); err != nil {
		return err
	}
	scan := bufio.NewScanner(r)
	scan.Split(bufio.ScanWords)

	s.locals = EnclosedEnv(s.locals)
	if u, ok := s.locals.(interface{ unwrap() Environment }); ok {
		defer func() {
			s.locals = u.unwrap()
		}()
	}
	for scan.Scan() {
		s.Define(cmd.ident, strArray(scan.Text()))
		if err := s.execute(cmd.body); err != nil {
			return err
		}
	}
	return e.Wait()
}

func (s *Shell) runCommand(words []string) error {
	e, err := s.lookupCommand(words[0], words[1:])
	if err != nil {
		return err
	}
	e.replaceIn(s.Stdin)
	e.replaceErr(s.Stderr)
	e.replaceOut(s.Stdout)
	return e.Run()
}

func (s *Shell) runBuiltin(words []string) error {
	e, err := s.lookupBuiltin(words[0], words[1:])
	if err != nil {
		return err
	}
	e.replaceIn(s.Stdin)
	e.replaceErr(s.Stderr)
	e.replaceOut(s.Stdout)
	return e.Run()
}

func (s *Shell) prepare(c Command) (Executable, error) {
	words, err := c.Expand(s)
	if err != nil {
		return nil, err
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("no command given")
	}
	s.exec.cmd = words[0]
	s.exec.args = words[1:]
	cmd, err := s.lookup(words[0], words[1:])
	if err != nil {
		return nil, err
	}
	cmd.replaceIn(s.Stdin)
	cmd.replaceOut(s.Stdout)
	cmd.replaceErr(s.Stderr)
	return cmd, nil
}

func (s *Shell) lookupBuiltin(cmd string, args []string) (Executable, error) {
	if b, ok := s.builtins[cmd]; ok {
		if b.Disabled {
			return nil, fmt.Errorf("%s is disabled", cmd)
		}
		e := b
		e.Shell = s
		e.Args = append(e.Args[:0], args...)
		return &e, nil
	}
	return nil, fmt.Errorf("%s builtin not found", cmd)
}

func (s *Shell) lookupCommand(cmd string, args []string) (Executable, error) {
	exists := func(path string) bool {
		s, err := os.Stat(path)
		return err == nil && s.Mode().IsRegular()
	}
	var env []string
	if i, ok := s.env.(interface{ List() []string }); ok {
		env = i.List()
	}
	if exists(cmd) {
		return External(cmd, args, env, s.WorkDir()), nil
	}
	for _, d := range s.path {
		d = filepath.Join(d, cmd)
		if exists(d) {
			return External(d, args, env, s.WorkDir()), nil
		}
		for _, e := range s.exext {
			if filepath.Ext(d) != e {
				if exists(d + e) {
					return External(d+e, args, env, s.WorkDir()), nil
				}
			}
		}
	}
	return External(cmd, args, env, s.WorkDir()), nil
}

func (s *Shell) lookup(cmd string, args []string) (Executable, error) {
	if e, err := s.lookupBuiltin(cmd, args); err == nil {
		return e, err
	}
	return s.lookupCommand(cmd, args)
}

func (s *Shell) Define(ident string, values []string) {
	s.locals.Define(ident, values)
}

func (s *Shell) Resolve(ident string) ([]string, error) {
	switch ident {
	case "?":
		n := strconv.Itoa(s.exec.code)
		return strArray(n), nil
	case "#":
		n := strconv.Itoa(len(s.exec.args))
		return strArray(n), nil
	case "@":
		return slices.Clone(s.exec.args), nil
	case "HOME":
		return strArray(""), nil
	case "PWD":
		return strArray(s.WorkDir()), nil
	case "OLDPWD":
		return strArray(s.OldWorkDir()), nil
	case "PID":
		n := os.Getpid()
		return strArray(strconv.Itoa(n)), nil
	case "PPID":
		n := os.Getppid()
		return strArray(strconv.Itoa(n)), nil
	case "PATH":
		return slices.Clone(s.path), nil
	case "RANDOM":
		n := rand.Int()
		return strArray(strconv.Itoa(n)), nil
	case "SECONDS":
		n := s.when.Unix()
		return strArray(strconv.Itoa(int(n))), nil
	case "SHELL":
		return strArray(tishName), nil
	case "VERSION":
		return strArray(tishVersion), nil
	default:
	}
	if n, err := strconv.Atoi(ident); err == nil && n >= 0 {
		if n > len(s.exec.args) {
			return strArray(""), nil
		}
		if n == 0 {
			return strArray(s.exec.cmd), nil
		}
		return strArray(s.exec.args[n]), nil
	}
	list, err := s.locals.Resolve(ident)
	if err != nil {
		list, err = s.env.Resolve(ident)
	}
	return list, err
}
