package tish

import (
	"errors"
	"fmt"
)

var (
	ErrExit  = errors.New("exit")
	ErrFalse = errors.New("false")
)

var builtins = map[string]builtin{
	// "echo": {
	// 	Usage: "echo [arg...]",
	// 	Call:  runEcho,
	// },
	"export": {
		Usage: "export",
		Call:  runExport,
	},
	"exit": {
		Usage: "exit",
		Call:  runExit,
	},
	"cd": {
		Usage: "cd",
		Call:  runCd,
	},
	"pwd": {
		Usage: "pwd",
		Call:  runPwd,
	},
	"builtin": {
		Usage: "builtin",
		Call:  runBuiltin,
	},
	"command": {
		Usage: "command",
		Call:  runCommand,
	},
	"enable": {
		Usage: "enable",
		Call:  runEnable,
	},
	"true": {
		Usage: "true",
		Call:  runTrue,
	},
	"false": {
		Usage: "false",
		Call:  runFalse,
	},
}

func runExit(b *builtin) error {
	return ErrExit
}

func runExport(b *builtin) error {
	return nil
}

func runEcho(b *builtin) error {
	for i := range b.Args {
		if i > 0 {
			fmt.Fprint(b.Stdout, " ")
		}
		fmt.Fprint(b.Stdout, b.Args[i])
	}
	fmt.Fprintln(b.Stdout)
	return nil
}

func runCd(b *builtin) error {
	return nil
}

func runPwd(b *builtin) error {
	fmt.Fprint(b.Stdout, b.Shell.WorkDir())
	fmt.Fprintln(b.Stdout)
	return nil
}

func runBuiltin(b *builtin) error {
	return nil
}

func runCommand(b *builtin) error {
	return nil
}

func runEnable(b *builtin) error {
	return nil
}

func runTrue(b *builtin) error {
	return nil
}

func runFalse(b *builtin) error {
	return ErrFalse
}
