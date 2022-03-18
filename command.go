package tish

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/midbel/tish/internal/stdio"
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
	c.Stdin = r
}

func (c *stdCommand) SetOut(w io.Writer) {
	c.Stdout = w
}

func (c *stdCommand) SetErr(w io.Writer) {
	c.Stderr = w
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

type Pipe struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	closes []io.Closer
}

func (p *Pipe) Setup() error {
	fset := []func() error {
		p.setStdin,
		p.setStdout,
		p.setStderr,
	}
	for _, f := range fset {
		err := f()
		if err != nil {
			p.Close()
			return err
		}
	}
	return nil
}

func (p *Pipe) Clear() {
	p.stdin = nil
	p.stdout = nil
	p.stderr = nil
	p.Reset()
}

func (p *Pipe) Reset() {
	p.closes = p.closes[:0]
}

func (p *Pipe) SetIn(r io.Reader) {
	p.stdin = r
}

func (p *Pipe) SetOut(w io.Writer) {
	p.stdout = w
}

func (p *Pipe) SetErr(w io.Writer) {
	p.stderr = w
}

func (p *Pipe) StdoutPipe() (io.ReadCloser, error) {
	if p.stdout != nil {
		return nil, fmt.Errorf("stdout already set")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	p.closes = append(p.closes, pr, pw)
	p.stdout = pw
	return pr, nil
}

func (p *Pipe) StderrPipe() (io.ReadCloser, error) {
	if p.stderr != nil {
		return nil, fmt.Errorf("stderr already set")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	p.closes = append(p.closes, pr, pw)
	p.stderr = pw
	return pr, nil
}

func (p *Pipe) StdinPipe() (io.WriteCloser, error) {
	if p.stdin != nil {
		return nil, fmt.Errorf("stdin already set")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	p.closes = append(p.closes, pr, pw)
	p.stdin = pr
	return pw, nil
}

func (p *Pipe) setStdin() error {
	if p.stdin == nil {
		f, err := os.Open(os.DevNull)
		if err != nil {
			return err
		}
		p.stdin = f
	}
	rc := stdio.Reader(p.stdin)
	p.closes = append(p.closes, rc)
	p.stdin = rc
	return nil
}

func (p *Pipe) setStdout() error {
	wc, err := p.openFile(p.stdout)
	if err == nil {
		p.stdout = wc
	}
	return err
}

func (p *Pipe) setStderr() error {
	wc, err := p.openFile(p.stderr)
	if err == nil {
		p.stderr = wc
	}
	return err
}

func (p *Pipe) openFile(w io.Writer) (io.WriteCloser, error) {
	if w == nil {
		f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			return nil, err
		}
		w = f
	}
	wc := stdio.Writer(w)
	p.closes = append(p.closes, wc)
	return wc, nil
}

func (p *Pipe) Close() error {
	for _, c := range p.closes {
		c.Close()
	}
	p.closes = p.closes[:0]
	return nil
}
