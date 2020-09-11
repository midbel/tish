package tish

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

type Builtin struct {
	Usage string
	Short string
	Desc  string

	Exit int
	*Shell

	Args []string
	Exec func(Builtin) int

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
		return nil
	}
	b.done = make(chan int, 1)
	go func() {
		b.done <- b.Exec(*b)
	}()
	return nil
}

func (b *Builtin) Wait() error {
	if !b.Runnable() {
		return nil
	}
	if b.finished {
		return nil
	}
	b.Exit = <-b.done

	b.finished = true
	close(b.done)
	if b.Exit != ExitOk {
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
}

func Echo(b Builtin) int {
	if exit := ParseArgs(b, nil); exit != 0 {
		return exit
	}
	fmt.Fprintln(b.Stdout, strings.Join(b.Args, " "))
	return ExitOk
}

func Exit(b Builtin) int {
	if exit := ParseArgs(b, nil); exit != 0 {
		return exit
	}
	return ExitOk
}

func Export(b Builtin) int {
	if exit := ParseArgs(b, nil); exit != 0 {
		return exit
	}
	return ExitOk
}

func True(b Builtin) int {
	if exit := ParseArgs(b, nil); exit != 0 {
		return exit
	}
	return ExitOk
}

func False(b Builtin) int {
	if exit := ParseArgs(b, nil); exit != 0 {
		return exit
	}
	return ExitKo
}

func ParseArgs(b Builtin, fn func(set *flag.FlagSet)) int {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	if fn != nil {
		fn(set)
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	return ExitOk
}
