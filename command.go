package tish

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type CommandFinder interface {
	Find(context.Context, string) (Command, error)
}

type CommandType int8

const (
	TypeBuiltin CommandType = iota
	TypeScript
	TypeExternal
	TypeRegular
)

type Command interface {
	Command() string
	Type() CommandType

	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)

	SetIn(r io.Reader)
	SetOut(w io.Writer)
	SetErr(w io.Writer)

	Run() error
	Start() error
	Wait() error
	Exit() (int, int)
}

type stdCommand struct {
	*exec.Cmd
	name string
}

func StandardContext(ctx context.Context, name, cwd string, args []string) Command {
	c := exec.CommandContext(ctx, name, args...)
	c.Dir = cwd
	return &stdCommand{
		Cmd:  c,
		name: name,
	}
}

func (c *stdCommand) Command() string {
	return c.name
}

func (_ *stdCommand) Type() CommandType {
	return TypeRegular
}

func (c *stdCommand) SetIn(r io.Reader) {
	c.Cmd.Stdin = r
}

func (c *stdCommand) SetOut(w io.Writer) {
	c.Cmd.Stdout = w
}

func (c *stdCommand) SetErr(w io.Writer) {
	c.Cmd.Stderr = w
}

func (c *stdCommand) Exit() (int, int) {
	if c == nil || c.Cmd == nil || c.Cmd.ProcessState == nil {
		return 0, 255
	}
	var (
		pid  = c.ProcessState.Pid()
		code = c.ProcessState.ExitCode()
	)
	return pid, code
}

func (c *stdCommand) SetEnv(env []string) {
	c.Cmd.Env = append(c.Cmd.Env[:0], env...)
}

type Builtin struct {
	Usage    string
	Short    string
	Help     string
	Disabled bool
	Execute  func(Builtin) error

	Args     []string
	shell    *Shell
	finished bool
	code     int
	done     chan error

	// Pipe
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
	closes []io.Closer

	errch chan error
}

func (b *Builtin) SetOut(w io.Writer) {
	b.Stdout = w
}

func (b *Builtin) SetErr(w io.Writer) {
	b.Stderr = w
}

func (b *Builtin) SetIn(r io.Reader) {
	b.Stdin = r
}

func (b *Builtin) StdinPipe() (io.WriteCloser, error) {
	if b.Stdin != nil {
		return nil, fmt.Errorf("builtin: stdin already set")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	b.SetIn(pr)
	b.closes = append(b.closes, pr)
	return pw, nil
}

func (b *Builtin) StdoutPipe() (io.ReadCloser, error) {
	if b.Stdout != nil {
		return nil, fmt.Errorf("builtin: stdout already set")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	b.SetOut(pw)
	b.closes = append(b.closes, pw)
	return pr, nil
}

func (b *Builtin) StderrPipe() (io.ReadCloser, error) {
	if b.Stderr != nil {
		return nil, fmt.Errorf("builtin: stderr already set")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	b.SetErr(pw)
	b.closes = append(b.closes, pw)
	return pr, nil
}

func (b *Builtin) Name() string {
	i := strings.Index(b.Usage, " ")
	if i <= 0 {
		return b.Usage
	}
	return b.Usage[:i]
}

func (b *Builtin) Command() string {
	return b.Name()
}

func (b *Builtin) IsEnabled() bool {
	return !b.Disabled && b.Execute != nil
}

func (b *Builtin) Exit() (int, int) {
	return 0, b.code
}

func (b *Builtin) Type() CommandType {
	return TypeBuiltin
}

func (b *Builtin) Start() error {
	if !b.IsEnabled() {
		return fmt.Errorf("builtin is disabled")
	}
	if b.finished {
		return fmt.Errorf("builtin already executed")
	}

	b.done = make(chan error, 1)
	go func() {
		b.done <- b.Execute(*b)
	}()
	return nil
}

func (b *Builtin) Wait() error {
	if !b.IsEnabled() {
		return fmt.Errorf("builtin is disabled")
	}
	if b.finished {
		return fmt.Errorf("builtin already finished")
	}
	b.finished = true
	err := <-b.done
	close(b.done)
	b.closeIO()

	if err != nil {
		b.code = 1
		return err
	}
	return nil
}

func (b *Builtin) Run() error {
	if err := b.Start(); err != nil {
		return err
	}
	return b.Wait()
}

func (b *Builtin) closeIO() error {
	for _, c := range b.closes {
		c.Close()
	}
	b.closes = b.closes[:0]
	return nil
}
