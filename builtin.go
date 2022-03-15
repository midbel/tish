package tish

import (
	"context"
	"flag"
	"fmt"
	// "io"
	// "os"
	"plugin"
	"strconv"
	"strings"
)

var builtins = map[string]Builtin{
	"set": {
		Usage:   "set",
		Short:   "set specific shell option",
		Help:    "",
		Execute: nil,
	},
	"echo": {
		Usage:   "echo",
		Short:   "echo the string(s) to standard output",
		Help:    "",
		Execute: runEcho,
	},
	"help": {
		Usage:   "help <builtin>",
		Short:   "display information about a builtin command",
		Help:    "",
		Execute: runHelp,
	},
	"builtins": {
		Usage:   "builtins",
		Short:   "display a list of supported builtins",
		Help:    "",
		Execute: runBuiltins,
	},
	"true": {
		Usage:   "true",
		Short:   "always return a successful result",
		Help:    "",
		Execute: runTrue,
	},
	"false": {
		Usage:   "false",
		Short:   "always return an unsuccessful result",
		Help:    "",
		Execute: runFalse,
	},
	"builtin": {
		Usage:   "builtin",
		Short:   "execute a simple builtin or display information about builtins",
		Help:    "",
		Execute: runBuiltin,
	},
	"command": {
		Usage:   "command",
		Short:   "execute a simple command or display information about commands",
		Help:    "",
		Execute: runCommand,
	},
	"seq": {
		Usage:   "seq [first] [inc] <last>",
		Short:   "print a sequence of number to stdout",
		Help:    "",
		Execute: runSeq,
	},
	"type": {
		Usage:   "type <name...>",
		Short:   "display information about command type",
		Help:    "",
		Execute: runType,
	},
	"env": {
		Usage:   "env",
		Short:   "display list of variables exported to environment of commands to be executed",
		Help:    "",
		Execute: runEnv,
	},
	"export": {
		Usage:   "export [name[=value]]...",
		Short:   "mark variables to export in environment of commands to be executed",
		Help:    "",
		Execute: runExport,
	},
	"enable": {
		Usage:   "enable [-p] [-d] [-f] <builtin...>",
		Short:   "enable and disable builtins",
		Help:    "",
		Execute: runEnable,
	},
	"alias": {
		Usage:   "alias [name[=value]]...",
		Short:   "define or display aliases",
		Help:    "",
		Execute: runAlias,
	},
	"unalias": {
		Usage:   "unalias [name...]",
		Short:   "remove each name from the list of defined aliases",
		Help:    "",
		Execute: runUnalias,
	},
	"cd": {
		Usage:   "cd <dir>",
		Short:   "change the shell working directory",
		Help:    "",
		Execute: runChdir,
	},
	"pwd": {
		Usage:   "pwd",
		Short:   "print the name of the current shell working directory",
		Help:    "",
		Execute: runPwd,
	},
	"popd": {
		Usage:   "popd",
		Short:   "",
		Help:    "",
		Execute: runPopd,
	},
	"pushd": {
		Usage:   "pushd",
		Short:   "",
		Help:    "",
		Execute: runPushd,
	},
	"dirs": {
		Usage:   "dirs",
		Short:   "",
		Help:    "",
		Execute: runDirs,
	},
	"readonly": {
		Usage:   "readonly <name>",
		Short:   "mark and unmark shell variables as readonly",
		Help:    "",
		Execute: runReadOnly,
	},
	"exit": {
		Usage:   "exit [code]",
		Short:   "exit the shell",
		Help:    "",
		Execute: runExit,
	},
	"wait": {
		Usage:   "wait [n]",
		Short:   "wait for process running in background",
		Help:    "",
		Execute: runWait,
	},
}

type Builtin struct {
	Usage    string
	Short    string
	Help     string
	Disabled bool
	Execute  func(Builtin) error

	args     []string
	shell    *Shell
	finished bool
	code     int
	done     chan error

	StdPipe
	errch chan error
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
	for _, set := range b.SetupFd() {
		_, err := set()
		if err != nil {
			b.Close()
			return err
		}
	}
	if copies := b.Copies(); len(copies) > 0 {
		b.errch = make(chan error, 3)
		for _, fn := range copies {
			go func(fn func() error) {
				b.errch <- fn()
			}(fn)
		}
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

	var (
		errex = <-b.done
		errcp error
	)
	defer close(b.done)
	b.Close()
	for range b.Copies() {
		e := <-b.errch
		if errcp == nil && e != nil {
			b.code = 2
			errcp = e
		}
	}
	if errex != nil {
		b.code = 1
		return errex
	}
	return errcp
}

func (b *Builtin) Run() error {
	if err := b.Start(); err != nil {
		return err
	}
	return b.Wait()
}

func runEcho(b Builtin) error {
	var (
		set   flag.FlagSet
		delim = set.String("d", " ", "strings delimiter")
	)
	if err := set.Parse(b.args); err != nil {
		return err
	}
	for i, a := range set.Args() {
		if i > 0 {
			fmt.Fprint(b.stdout, *delim)
		}
		fmt.Fprint(b.stdout, a)
	}
	fmt.Fprintln(b.stdout)
	return nil
}

func runTrue(_ Builtin) error {
	return nil
}

func runFalse(_ Builtin) error {
	return Failure
}

func runBuiltins(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	for n, i := range b.shell.builtins {
		if i.Name() != "" {
			n = i.Name()
		}
		fmt.Fprintf(b.stdout, "%-12s: %s", n, i.Short)
		fmt.Fprintln(b.stdout)
	}
	return nil
}

func runHelp(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	other, ok := b.shell.builtins[set.Arg(0)]
	if !ok {
		fmt.Fprintf(b.stderr, "no help match %s! try builtins to get the list of available builtins", set.Arg(0))
		fmt.Fprintln(b.stderr)
		return nil
	}
	fmt.Fprintln(b.stdout, other.Name())
	fmt.Fprintln(b.stdout, other.Short)
	fmt.Fprintln(b.stdout)
	if len(other.Help) > 0 {
		fmt.Fprintln(b.stdout, other.Help)
	}
	fmt.Fprintln(b.stdout)
	return nil
}

func runBuiltin(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	if set.NArg() == 0 {
		fmt.Fprintln(b.stderr, "not enough argument supplied")
		return nil
	}
	other, ok := b.shell.builtins[set.Arg(0)]
	if !ok {
		fmt.Fprintf(b.stderr, "%s: unknown builtin", set.Arg(0))
		fmt.Fprintln(b.stderr)
		return nil
	}
	for i := 1; i < set.NArg(); i++ {
		other.args = append(other.args, set.Arg(i))
	}
	other.shell = b.shell
	other.stdout = b.stdout
	other.stderr = b.stderr
	other.stdin = b.stdin

	return other.Run()
}

func runCommand(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	return nil
}

func runType(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	for _, a := range set.Args() {
		var kind string
		if _, ok := b.shell.builtins[a]; ok {
			kind = "builtin"
		} else if _, err := b.shell.Find(context.TODO(), a); err == nil {
			kind = "user command"
		} else if _, ok := b.shell.alias[a]; ok {
			kind = "alias"
		} else if vs, err := b.shell.Resolve(a); err == nil && len(vs) > 0 {
			kind = "shell variable"
		} else {
			kind = "command"
		}
		fmt.Fprintf(b.stdout, "%s is %s", a, kind)
		fmt.Fprintln(b.stdout)
	}
	return nil
}

func runSeq(b Builtin) error {
	var (
		set flag.FlagSet
		sep = set.String("s", " ", "print separator between each number")
		fst = 1
		lst = 1
		inc = 1
		err error
	)
	if err := set.Parse(b.args); err != nil {
		return err
	}
	switch set.NArg() {
	case 1:
		if lst, err = strconv.Atoi(set.Arg(0)); err != nil {
			fmt.Fprintf(b.stderr, "%s: invalid number", flag.Arg(0))
			fmt.Fprintln(b.stderr)
		}
	case 2:
		if fst, err = strconv.Atoi(set.Arg(0)); err != nil {
			fmt.Fprintf(b.stderr, "%s: invalid number", flag.Arg(0))
			fmt.Fprintln(b.stderr)
			break
		}
		if lst, err = strconv.Atoi(set.Arg(1)); err != nil {
			fmt.Fprintf(b.stderr, "%s: invalid number", flag.Arg(1))
			fmt.Fprintln(b.stderr)
			break
		}
	case 3:
		if fst, err = strconv.Atoi(set.Arg(0)); err != nil {
			fmt.Fprintf(b.stderr, "%s: invalid number", flag.Arg(0))
			fmt.Fprintln(b.stderr)
			break
		}
		if inc, err = strconv.Atoi(set.Arg(1)); err != nil {
			fmt.Fprintf(b.stderr, "%s: invalid number", flag.Arg(1))
			fmt.Fprintln(b.stderr)
			break
		}
		if lst, err = strconv.Atoi(set.Arg(2)); err != nil {
			fmt.Fprintf(b.stderr, "%s: invalid number", flag.Arg(2))
			fmt.Fprintln(b.stderr)
			break
		}
	default:
		fmt.Fprintf(b.stderr, "seq: missing operand")
		fmt.Fprintln(b.stderr)
		return nil
	}
	if err != nil {
		return nil
	}
	if inc == 0 {
		inc++
	}
	cmp := func(f, t int) bool { return f <= t }
	if fst > lst {
		cmp = func(f, t int) bool { return f >= t }
		if inc > 0 {
			inc = -inc
		}
	}
	for i := 0; cmp(fst, lst); i++ {
		if i > 0 {
			fmt.Fprint(b.stdout, *sep)
		}
		fmt.Fprintf(b.stdout, strconv.Itoa(fst))
		fst += inc
	}
	fmt.Fprintln(b.stdout)
	return nil
}

func runEnable(b Builtin) error {
	var set flag.FlagSet
	var (
		print   = set.Bool("p", false, "print the list of builtins with their status")
		load    = set.Bool("f", false, "load new builtin(s) from list of given object file(s)")
		disable = set.Bool("d", false, "disable builtin(s) given in the list")
	)
	if err := set.Parse(b.args); err != nil {
		return err
	}
	if *load {
		return loadExternalBuiltins(b, set.Args())
	}
	if *print {
		printEnableBuiltins(b)
		return nil
	}
	for _, n := range set.Args() {
		other, ok := b.shell.builtins[n]
		if !ok {
			fmt.Fprintf(b.stderr, "builtin %s not found", n)
			fmt.Fprintln(b.stderr)
			continue
		}
		other.Disabled = *disable
		b.shell.builtins[n] = other
	}
	return nil
}

func loadExternalBuiltins(b Builtin, files []string) error {
	for _, f := range files {
		plug, err := plugin.Open(f)
		if err != nil {
			return err
		}
		sym, err := plug.Lookup("Load")
		if err != nil {
			return err
		}
		load, ok := sym.(func() Builtin)
		if !ok {
			return fmt.Errorf("invalid signature")
		}
		e := load()
		b.shell.builtins[b.Name()] = e
	}
	return nil
}

func printEnableBuiltins(b Builtin) {
	for _, x := range b.shell.builtins {
		state := "enabled"
		if x.Disabled {
			state = "disabled"
		}
		fmt.Fprintf(b.stdout, "%-12s: %s", x.Name(), state)
		fmt.Fprintln(b.stdout)
	}
}

func runReadOnly(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	return nil
}

func runEnv(b Builtin) error {
	for n, v := range b.shell.env {
		fmt.Fprintf(b.stdout, "%-10s = %s", n, v)
		fmt.Fprintln(b.stdout)
	}
	return nil
}

func runExport(b Builtin) error {
	var (
		set flag.FlagSet
		del = set.Bool("d", false, "delete")
	)
	if err := set.Parse(b.args); err != nil {
		return err
	}
	for _, k := range set.Args() {
		if *del {
			b.shell.Unexport(k)
			continue
		}
		var v string
		if x := strings.Index(k, "="); x > 0 {
			k, v = k[:x], v[x+1:]
		}
		b.shell.Export(k, v)
	}
	return nil
}

func runExit(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	code := ExitCode(b.shell.context.code)
	if c, err := strconv.Atoi(set.Arg(0)); err == nil {
		code = ExitCode(c)
	}
	if code.Failure() {
		return fmt.Errorf("%w: %s", ErrExit, code)
	}
	return nil
}

func runChdir(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	if err := b.shell.Chdir(set.Arg(0)); err != nil {
		fmt.Fprintf(b.stderr, err.Error())
		fmt.Fprintln(b.stderr)
	}
	return nil
}

func runPwd(b Builtin) error {
	fmt.Fprintln(b.stdout, b.shell.Cwd())
	return nil
}

func runPopd(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	return b.shell.Popd(set.Arg(0))
}

func runPushd(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	return b.shell.Pushd(set.Arg(0))
}

func runDirs(b Builtin) error {
	var (
		set    flag.FlagSet
		clear  = set.Bool("c", false, "clear directory stack")
		line   = set.Bool("p", false, "print one entry per line")
		prefix = set.Bool("v", false, "print prefix")
	)
	if err := set.Parse(b.args); err != nil {
		return err
	}
	if *clear {

	}
	eol := " "
	if *line || *prefix {
		eol = "\n"
	}
	for i, d := range b.shell.Dirs() {
		if i > 0 {
			fmt.Fprint(b.stdout, eol)
		}
		if *prefix {
			fmt.Fprintf(b.stdout, "%d ", i+1)
		}
		fmt.Fprint(b.stdout, d)
	}
	fmt.Fprintln(b.stdout)
	return nil
}

func runAlias(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	if set.NArg() == 0 {
		for k, a := range b.shell.alias {
			fmt.Fprintf(b.stdout, "%s: %s", k, strings.Join(a, " "))
			fmt.Fprintln(b.stdout)
		}
	}
	for _, k := range set.Args() {
		var v string
		if x := strings.Index(k, "="); x > 0 {
			k, v = k[:x], v[x+1:]
		}
		b.shell.Alias(k, v)
	}
	return nil
}

func runUnalias(b Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.args); err != nil {
		return err
	}
	for _, a := range set.Args() {
		b.shell.Unalias(a)
	}
	return nil
}

func runWait(b Builtin) error {
	return nil
}
