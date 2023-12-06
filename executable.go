package tish

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
)

type Executable interface {
	Run() error
	Start() error
	Wait() error

	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)

	replaceIn(io.Reader)
	replaceOut(io.Writer)
	replaceErr(io.Writer)
}

type external struct {
	*exec.Cmd
}

func External(cmd string, args, env []string, workdir string) Executable {
	c := exec.Command(cmd, args...)
	c.Env = env
	c.Dir = workdir
	return &external{
		Cmd: c,
	}
}

func (e *external) replaceIn(r io.Reader) {
	e.Cmd.Stdin = r
}

func (e *external) replaceOut(w io.Writer) {
	e.Cmd.Stdout = w
}

func (e *external) replaceErr(w io.Writer) {
	e.Cmd.Stderr = w
}

type builtin struct {
	Usage string
	Call  func(*builtin) error

	*Shell
	Disabled bool
	Args     []string
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer

	code     int
	finished bool
	done     chan error

	writers []io.Closer
	readers []io.Closer
}

func (b *builtin) Start() error {
	if b.Disabled {
		return fmt.Errorf("builtin disabled")
	}
	if b.finished || b.done != nil {
		return fmt.Errorf("builtin already executed")
	}

	if b.Stdout == nil {
		b.Stdout = io.Discard
	}
	if b.Stderr == nil {
		b.Stderr = io.Discard
	}

	b.done = make(chan error, 1)
	go func() {
		b.done <- b.Call(b)
		b.closeWriters()
	}()
	return nil
}

func (b *builtin) Wait() error {
	if b.Disabled {
		return fmt.Errorf("builtin disabled")
	}
	if b.done == nil {
		return fmt.Errorf("builtin not yet started")
	}
	if b.finished {
		return fmt.Errorf("builtin already finished")
	}
	b.finished = true
	err := <-b.done
	close(b.done)
	b.closeReaders()

	if err != nil {
		b.code = 1
	}
	if errors.Is(err, ErrFalse) {
		err = nil
	}
	return err
}

func (b *builtin) Run() error {
	if err := b.Start(); err != nil {
		return err
	}
	return b.Wait()
}

func (b *builtin) StdoutPipe() (io.ReadCloser, error) {
	if b.Stdout != nil {
		return nil, fmt.Errorf("stdout already set")
	}
	pr, pw := io.Pipe()
	b.Stdout = pw
	b.writers = append(b.writers, pw)
	b.readers = append(b.readers, pr)
	return pr, nil
}

func (b *builtin) StderrPipe() (io.ReadCloser, error) {
	if b.Stderr != nil {
		return nil, fmt.Errorf("stderr already set")
	}
	pr, pw := io.Pipe()
	b.Stderr = pw
	b.writers = append(b.writers, pw)
	b.readers = append(b.readers, pr)
	return pr, nil
}

func (b *builtin) replaceIn(r io.Reader) {
	b.Stdin = r
}

func (b *builtin) replaceOut(w io.Writer) {
	b.Stdout = w
}

func (b *builtin) replaceErr(w io.Writer) {
	b.Stderr = w
}

func (b *builtin) closeWriters() {
	for i := range b.writers {
		b.writers[i].Close()
	}
}

func (b *builtin) closeReaders() {
	for i := range b.readers {
		b.readers[i].Close()
	}
}
