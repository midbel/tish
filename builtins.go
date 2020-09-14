package tish

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Builtin struct {
	Flag flag.FlagSet

	Usage string
	Short string
	Desc  string

	*Shell

	Args []string
	Exec func(Builtin) int

	exit     int
	done     chan int
	finished bool

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (b *Builtin) String() string {
	if i := strings.Index(b.Usage, " "); i > 0 {
		return b.Usage[:i]
	}
	return b.Usage
}

func (b *Builtin) Help() string {
	return ""
}

func (b *Builtin) Runnable() bool {
	return b.Exec != nil
}

func (b *Builtin) Start() error {
	if !b.Runnable() {
		return fmt.Errorf("%s: can not be executed", b.String())
	}
	if b.finished {
		return fmt.Errorf("%s: already done", b.String())
	}
	b.done = make(chan int, 1)
	go func() {
		b.done <- b.Exec(*b)
	}()
	return nil
}

func (b *Builtin) Wait() error {
	if !b.Runnable() {
		return fmt.Errorf("%s: can not be executed", b.String())
	}
	if b.finished {
		return fmt.Errorf("%s: already done", b.String())
	}

	b.exit = <-b.done
	b.finished = true
	close(b.done)
	if b.exit != ExitOk {
		return fmt.Errorf("%s: fail to execute properly", b.String())
	}
	return nil
}

func (b *Builtin) Run() error {
	if err := b.Start(); err != nil {
		return err
	}
	return b.Wait()
}

func (b *Builtin) Execute() Status {
	err := b.Run()
	return Status{
		Exit: b.exit,
		Pid:  b.Shell.pid,
		Err:  err,
	}
}

func (b *Builtin) Close() error {
	if c, ok := b.Stdin.(io.Closer); ok {
		c.Close()
	}
	if c, ok := b.Stdout.(io.Closer); ok {
		c.Close()
	}
	if c, ok := b.Stderr.(io.Closer); ok {
		c.Close()
	}
	return nil
}

func (b *Builtin) ParseArgs(fn func()) int {
	b.Flag.Init(b.String(), flag.ContinueOnError)
	b.Flag.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	help := b.Flag.Bool("h", false, "show help message and exit")
	if fn != nil {
		fn()
	}
	if err := b.Flag.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitUsage
	}
	if *help {
		b.Flag.Usage()
		return ExitHelp
	}
	return ExitOk
}

var builtins = map[string]Builtin{
	"echo": {
		Usage: "echo [string...]",
		Short: "echo the given string(s) to stdout",
		Exec:  Echo,
	},
	"exit": {
		Usage: "exit [code]",
		Short: "exit causes to exit the shell with the given code",
		Exec:  Exit,
	},
	"export": {
		Usage: "export ident[[=value]]",
		Short: "export the given value in the environment",
		Exec:  Export,
	},
	"true": {
		Usage: "true",
		Short: "true always returns a success result",
		Exec:  True,
	},
	"false": {
		Usage: "false",
		Short: "false always returns an unsuccessfull result",
		Exec:  False,
	},
	"builtin": {
		Usage: "builtin <cmd> [args]",
		Short: "run a builtin given its arguments",
		Exec:  ExecBuiltin,
	},
	"command": {
		Usage: "command <cmd> [args]",
		Short: "run a command given its arguments",
		Exec:  ExecCommand,
	},
	"alias": {
		Usage: "alias [-p] name[[=value]]",
		Short: "define an alias for each name given with a value",
		Exec:  Alias,
	},
	"unalias": {
		Usage: "unalias [name...]",
		Short: "remove each name from the list of alias",
		Exec:  Unalias,
	},
	"seq": {
		Usage: "seq [lower] [upper] [increment]",
		Short: "print a sequence of number",
		Exec:  Seq,
	},
}

func Seq(b Builtin) int {
	return ExitOk
}

func Alias(b Builtin) int {
	return ExitOk
}

func Unalias(b Builtin) int {
	if exit := b.ParseArgs(nil); exit != ExitOk {
		return exit
	}
	b.Shell.UnregisterAlias(b.Flag.Args()...)
	return ExitOk
}

func Echo(b Builtin) int {
	if exit := b.ParseArgs(nil); exit != 0 {
		return exit
	}
	_, err := fmt.Fprintln(b.Stdout, strings.Join(b.Flag.Args(), " "))
	if err != nil {
		return ExitKo
	}
	return ExitOk
}

func Exit(b Builtin) int {
	if exit := b.ParseArgs(nil); exit != 0 {
		return exit
	}
	if b.Flag.NArg() == 0 {
		return ExitQuit + b.Shell.proc.exit
	}
	if n, err := strconv.Atoi(b.Flag.Arg(0)); err == nil {
		if n < 0 {
			return ExitQuit + ExitKo
		}
		return ExitQuit + n
	}
	return ExitQuit
}

func Export(b Builtin) int {
	if exit := b.ParseArgs(nil); exit != 0 {
		return exit
	}
	return ExitOk
}

func ExecBuiltin(b Builtin) int {
	return ExitOk
}

func ExecCommand(b Builtin) int {
	return ExitOk
}

func True(b Builtin) int {
	if exit := b.ParseArgs(nil); exit != 0 {
		return exit
	}
	return ExitOk
}

func False(b Builtin) int {
	if exit := b.ParseArgs(nil); exit != 0 {
		return exit
	}
	return ExitKo
}
