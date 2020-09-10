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

	Args []string
	Exec func(Builtin) int

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (b Builtin) String() string {
  return ""
}

func (b Builtin) Runnable() bool {
	return b.Exec != nil
}

func (b Builtin) Start() error {
	return nil
}

func (b Builtin) Wait() error {
	return nil
}

func (b Builtin) Run() error {
	return nil
}

var builtins = map[string]*Builtin{
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
