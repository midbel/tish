package tish

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"slices"
)

var (
	ErrExit  = errors.New("exit")
	ErrFalse = errors.New("false")
)

var builtins = map[string]builtin{
	"alias": {
		Usage: "alias [name[=value]...] ",
		Call:  runAlias,
	},
	"unalias": {
		Usage: "unalias [name...]",
		Call:  runUnalias,
	},
	"type": {
		Usage: "type name",
		Call:  runType,
	},
	"printf": {
		Usage: "printf",
		Call:  runPrintf,
	},
	"echo": {
		Usage: "echo [arg...]",
		Call:  runEcho,
	},
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
	"pushd": {
		Usage: "pushd",
		Call:  runPushd,
	},
	"popd": {
		Usage: "popd",
		Call:  runPopd,
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
	"readonly": {
		Usage: "readonly",
		Call:  runReadOnly,
	},
}

func runReadOnly(b *builtin) error {
	var (
		set   = flag.NewFlagSet("alias", flag.ExitOnError)
		print = flag.Bool("p", false, "print registered alias")
	)
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *print {
		return nil
	}
	return nil
}

func runAlias(b *builtin) error {
	var (
		set   = flag.NewFlagSet("alias", flag.ExitOnError)
		print = flag.Bool("p", false, "print registered alias")
	)
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *print {
		return nil
	}
	b.Shell.alias[set.Arg(0)] = slices.Clone(set.Args()[1:])
	return nil
}

func runUnalias(b *builtin) error {
	return nil
}

func runType(b *builtin) error {
	set := flag.NewFlagSet("type", flag.ExitOnError)
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	for _, a := range set.Args() {
		if _, err := b.Shell.lookupBuiltin(a, nil); err == nil {
			fmt.Fprintln(b.Stdout, "builtin")
		} else if _, err := b.Shell.lookupCommand(a, nil); err == nil {
			fmt.Fprintln(b.Stdout, "command")
		} else if _, err := b.Shell.lookupAlias(a, nil); err == nil {
			fmt.Fprintln(b.Stdout, "alias")
		} else if _, err := os.Stat(a); err == nil {
			fmt.Fprintln(b.Stdout, "file")
		} else {
			fmt.Fprintln(b.Stderr, "unknown")
		}
	}
	return nil
}

func runPrintf(b *builtin) error {
	return nil
}

func runExit(b *builtin) error {
	return ErrExit
}

func runExport(b *builtin) error {
	for i := 0; i < len(b.Args); i += 2 {
		var (
			id = b.Args[i]
			vl = b.Args[i+1]
		)
		b.Shell.setEnv(id, []string{vl})
	}
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
	set := flag.NewFlagSet("cd", flag.ContinueOnError)
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if set.NArg() == 0 {
		home, err := b.getEnv(HomeEnv)
		if err != nil {
			return err
		}
		if err = os.Chdir(home[0]); err == nil {
			b.Shell.setCwd(home[0])
		}
		return err
	}
	if filepath.IsAbs(set.Arg(0)) {
		err := os.Chdir(set.Arg(0))
		if err == nil {
			b.Shell.setCwd(set.Arg(0))
		}
		return err
	}
	var (
		path = filepath.Join(b.Shell.WorkDir(), set.Arg(0))
		err  = os.Chdir(path)
	)
	if err == nil {
		b.Shell.setCwd(path)
		return nil
	}
	for _, d := range b.Shell.getPathCD() {
		path = filepath.Join(d, set.Arg(0))
		if err = os.Chdir(path); err == nil {
			b.Shell.setCwd(path)
			break
		}
	}
	return err
}

func runPushd(b *builtin) error {
	return nil
}

func runPopd(b *builtin) error {
	return nil
}

func runPwd(b *builtin) error {
	fmt.Fprint(b.Stdout, b.Shell.WorkDir())
	fmt.Fprintln(b.Stdout)
	return nil
}

func runBuiltin(b *builtin) error {
	return b.runBuiltin(b.Args)
}

func runCommand(b *builtin) error {
	return b.runCommand(b.Args)
}

func runEnable(b *builtin) error {
	set := flag.NewFlagSet("enable", flag.ContinueOnError)
	var (
		disabled = set.Bool("n", false, "disable shell builtin")
		print    = set.Bool("p", false, "print list of enable builtin")
		all      = set.Bool("a", false, "print all builtins with an indicator of their status")
		load     = set.String("f", "", "filename")
	)
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	switch {
	case *load != "":
		p, err := plugin.Open(*load)
		if err != nil {
			return err
		}
		s, err := p.Lookup("")
		if err != nil {
			return err
		}
		_ = s
		return notImplemented("enable -f")
	case *print:
		for n, x := range b.Shell.builtins {
			if *all {
				if !x.Disabled {
					fmt.Fprint(b.Stdout, "* ")
				}
			} else {
				if x.Disabled {
					continue
				}
			}
			fmt.Fprintln(b.Stdout, n)
		}
	default:
		for _, n := range set.Args() {
			x, ok := b.Shell.builtins[n]
			if !ok {
				continue
			}
			x.Disabled = *disabled
			b.Shell.builtins[n] = x
		}
	}
	return nil
}

func runTrue(b *builtin) error {
	return nil
}

func runFalse(b *builtin) error {
	return ErrFalse
}

func notImplemented(ident string) error {
	return fmt.Errorf("%s not yet implemented", ident)
}
