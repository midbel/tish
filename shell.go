package tish

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"plugin"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

var (
	ErrFailed = errors.New("process terminated with failure")
	ErrFatal  = errors.New("fatal")
)

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

	options Option
}

type Option uint64

const (
	NoFileExpansion Option = 1 << (iota + 1)
	NoBraceExpansion
	NoOverwriteFiles
	ExitOnError
	NoLocalVariables
	AllowUndefinedVariables
	NullGlob
	EmptyGlob
)

const DefaultOptions = NoLocalVariables | AllowUndefinedVariables

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
		options:    DefaultOptions,
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
	case kindSub:
		execute = s.executeSubstitution
	default:
		return fmt.Errorf("tish: %s can not be executed", i.kind)
	}
	return execute(i.words)
}

func (s *Shell) executeAssignment(a Assignment) error {
	vs, err := a.Expand(s)
	if err == nil {
		err = s.Define(a.ident, vs)
	}
	return err
}

func (s *Shell) executeSequence(ws []Word) error {
	var err error
	for _, w := range ws {
		err = s.execute(w)
		if errors.Is(err, ErrFatal) {
			return err
		}
	}
	return err
}

func (s *Shell) executeOr(ws []Word) error {
	var err error
	for _, w := range ws {
		err = s.execute(w)
		if errors.Is(err, ErrFatal) {
			return err
		}
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
		if errors.Is(err, ErrFatal) {
			return err
		}
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
	errw := grp.Wait()
	if s.proc.exit.Failure() && s.ExitOnError() {
		return ErrFatal
	}
	return errw
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
	if s.proc.exit.Failure() && s.ExitOnError() {
		return ErrFatal
	}
	return nil
}

func (s *Shell) executeSubshell(ws []Word) error {
	if len(ws) != 1 {
		return ErrFailed
	}
	code, err := s.executeShell(ws[0], s.stdin, s.stdout, s.stderr)
	if err == nil && code.Failure() {
		err = ErrFailed
	}
	return err
}

func (s *Shell) executeSubstitution(ws []Word) error {
	args, err := s.expandSubstitution(ws)
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
	if s.proc.exit.Failure() && s.ExitOnError() {
		return ErrFatal
	}
	return nil
}

func (s *Shell) executeShell(w Word, in io.Reader, out, err io.Writer) (ErrCode, error) {
	var (
		fs = s.Filesystem.Copy()
		sh = NewShell(fs, in, out, err)
	)
	sh.locals = s.locals.Copy()
	sh.globals = sh.locals.Unwrap()
	sh.level = s.level + 1

	errx := sh.execute(w)
	return sh.proc.exit, errx
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
		var (
			xs  []string
			err error
		)
		switch w := w.(type) {
		case List:
			if w.kind == kindSub {
				xs, err = s.expandSubstitution(w.words)
			} else {
				xs, err = w.Expand(s)
			}
		default:
			if _, ok := w.(Brace); ok && !s.AllowBraceExpansion() {
				continue
			}
			xs, err = w.Expand(s)
		}
		if err != nil {
			return nil, err
		}
		args = append(args, xs...)
	}
	cmd, err := s.prepare(s.expandFilenames(args))
	if err != nil {
		return nil, err
	}
	return s.replaceFiles(cmd, rs)
}

func (s *Shell) expandSubstitution(ws []Word) ([]string, error) {
	var (
		out  bytes.Buffer
		word = List{
			kind:  kindSeq,
			words: ws,
		}
	)
	var (
		args []string
		err  error
		code ErrCode
	)
	if code, err = s.executeShell(word, s.stdin, &out, s.stderr); err == nil && code.Success() {
		args = Words(&out)
	}
	if code.Failure() {
		err = ErrFailed
	}
	return args, err
}

func (s *Shell) replaceFiles(cmd Command, rs []Redirect) (Command, error) {
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
			args, err := r.Expand(s)
			if err != nil {
				return nil, err
			}
			if len(args) == 0 {
				continue
			}
			var flag int
			switch r.mode {
			case modRead:
				flag = os.O_RDONLY
			case modWrite:
				if !s.AllowOverwritingFiles() {
					flag = os.O_EXCL
				}
				flag = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
			case modAppend:
				flag = os.O_CREATE | os.O_APPEND | os.O_WRONLY
			default:
				return nil, fmt.Errorf("unsupported mode")
			}
			f, err := s.OpenFile(args[0], flag, 0644)
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
	cmd, err := s.LookPath(args[0])
	if err != nil {
		return nil, err
	}
	c := exec.Command(cmd, args[1:]...)

	if es := s.Environ(); len(es) > 0 {
		c.Env = append(c.Env, es...)
	}
	c.Dir = s.Cwd()

	c.Stdin = s.stdin
	c.Stdout = s.stdout
	c.Stderr = s.stderr

	return &Cmd{Cmd: c}, nil
}

func (s *Shell) expandFilenames(args []string) []string {
	if !s.AllowFileExpansion() {
		return args
	}
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

func (s *Shell) Dirs() []string {
	var (
		dirs = make([]string, 0, 10)
		home string
	)
	vs, _ := s.Resolve("HOME")
	if len(vs) > 0 {
		home = vs[0]
	}
	for _, d := range s.Filesystem.Dirs() {
		if home != "" && strings.HasPrefix(d, home) {
			d = strings.Replace(d, home, "~", 1)
		}
		dirs = append(dirs, d)
	}
	return dirs
}

func (s *Shell) Chroot(root string) error {
	if root == "-" {
		if s.Filesystem.parent != nil {
			s.Filesystem = s.Filesystem.parent
		}
		return nil
	}
	fs, err := s.Filesystem.Chroot(root)
	if err != nil {
		return err
	}
	s.Filesystem = fs
	return nil
}

func (s *Shell) LookPath(cmd string) (string, error) {
	ps, _ := s.Resolve("PATH")
	return s.Filesystem.LookPath(cmd, ps)
}

func (s *Shell) Exit(n ErrCode) {
	os.Exit(int(n))
}

func (s *Shell) Extend(files []string, replace bool) error {
	for _, f := range files {
		p, err := plugin.Open(path.Join(s.cwd(), f))
		if err != nil {
			return err
		}
		sym, err := p.Lookup("Builtins")
		if err != nil {
			return fmt.Errorf("missing Builtins symbol")
		}
		list, ok := sym.(func() []*Builtin)
		if !ok {
			return fmt.Errorf("symbol: invalid signature")
		}
		for _, b := range list() {
			if _, ok := builtins[b.String()]; ok && !replace {
				continue
			}
			b.external = true
			builtins[b.String()] = *b
		}
	}
	return nil
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
		if errors.Is(err, ErrNotDefined) && s.AllowUndefinedVariables() {
			err = nil
		}
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
	if !s.CopyAllVariables() {
		es := s.globals.Environ()
		return append(es, s.LocalEnviron()...)
	}
	return s.locals.Environ()
}

func (s *Shell) LocalEnviron() []string {
	return s.locals.LocalEnviron()
}

func (s *Shell) CopyAllVariables() bool {
	return s.options&NoLocalVariables == 0
}

func (s *Shell) AllowOverwritingFiles() bool {
	return s.options&NoOverwriteFiles == 0
}

func (s *Shell) AllowFileExpansion() bool {
	return s.options&NoFileExpansion == 0
}

func (s *Shell) AllowBraceExpansion() bool {
	return s.options&NoBraceExpansion == 0
}

func (s *Shell) AllowUndefinedVariables() bool {
	return s.options&AllowUndefinedVariables == 0
}

func (s *Shell) ExitOnError() bool {
	return s.options&ExitOnError != 0
}

func (s *Shell) NullGlob() bool {
	return s.options&NullGlob != 0
}

func (s *Shell) EmptyGlob() bool {
	return s.options&EmptyGlob != 0
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
