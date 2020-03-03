package tish

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type ErrCode int

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

func (e ErrCode) Success() bool {
	return e == ExitOk || e == ExitHelp
}

func (e ErrCode) Failure() bool {
	return !e.Success()
}

type Command interface {
	Start() error
	Wait() ErrCode
	Run() ErrCode

	Replace(int, *os.File) error
	Copy(int, int)
}

type Cmd struct {
	*exec.Cmd
}

func (c *Cmd) Copy(src, dst int) {
	if src == dst {
		return
	}
	switch src {
	case fdOut:
		c.Stdout = c.Stderr
	case fdErr:
		c.Stderr = c.Stdout
	default:
	}
}

func (c *Cmd) Replace(fd int, f *os.File) error {
	switch fd {
	case fdIn:
		closeFile(c.Stdin)
		c.Stdin = f
	case fdOut:
		if err := sameFile(c.Stdin, f); err != nil {
			return err
		}
		closeFile(c.Stdout)
		c.Stdout = f
	case fdErr:
		if err := sameFile(c.Stdin, f); err != nil {
			return err
		}
		closeFile(c.Stderr)
		c.Stderr = f
	case fdBoth:
		if err := sameFile(c.Stdin, f); err != nil {
			return err
		}
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
