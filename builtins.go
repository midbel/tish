package tish

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"plugin"
	"strconv"
	"strings"
	"time"
)

type Builtin struct {
	Usage string
	Short string
	Desc  string

	*Shell

	disabled bool
	external bool

	Args []string
	Exec func(Builtin) ErrCode

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	finished bool
	done     chan ErrCode
}

func (b *Builtin) String() string {
	if i := strings.Index(b.Usage, " "); i > 0 {
		return b.Usage[:i]
	}
	return b.Usage
}

func (b *Builtin) Help() string {
	var buf strings.Builder
	buf.WriteString(b.Short)
	if b.Desc != "" {
		buf.WriteRune(newline)
		buf.WriteString(b.Desc)
	}
	buf.WriteRune(newline)
	buf.WriteString("usage: " + b.Usage)
	return buf.String()
}

func (b *Builtin) Runnable() bool {
	return !b.disabled && b.Exec != nil
}

func (b *Builtin) Start() error {
	if !b.Runnable() {
		b.closeDescriptors()
		return fmt.Errorf("%s: not runnable", b.String())
	}
	if b.finished {
		b.closeDescriptors()
		return fmt.Errorf("%s: already executed", b.String())
	}

	b.done = make(chan ErrCode, 1)
	go func() {
		if b.Stdin == nil {
			b.Stdin = b.Shell.stdin
		}
		if b.Stdout == nil {
			b.Stdout = b.Shell.stdout
		}
		if b.Stderr == nil {
			b.Stderr = b.Shell.stderr
		}
		b.done <- b.Exec(*b)
		close(b.done)
		b.closeDescriptors()
	}()
	return nil
}

func (b *Builtin) Wait() ErrCode {
	if !b.Runnable() {
		fmt.Fprintf(b.Stderr, "%s: not runnable\n", b.String())
		return ExitNotExec
	}
	if b.finished {
		fmt.Fprintf(b.Stderr, "%s: already executed\n", b.String())
	}
	b.finished = true

	return <-b.done
}

func (b *Builtin) Run() ErrCode {
	if err := b.Start(); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitExec
	}
	return b.Wait()
}

func (b *Builtin) Copy(src, dst int) {
	if src == dst {
		return
	}
	switch src {
	case fdOut:
		b.Stdout = b.Stderr
	case fdErr:
		b.Stderr = b.Stdout
	default:
	}
}

func (b *Builtin) Replace(fd int, f *os.File) error {
	switch fd {
	case fdIn:
		closeFile(b.Stdin)
		b.Stdin = f
	case fdOut:
		if err := sameFile(b.Stdin, f); err != nil {
			return err
		}
		closeFile(b.Stdout)
		b.Stdout = f
	case fdErr:
		if err := sameFile(b.Stdin, f); err != nil {
			return err
		}
		closeFile(b.Stderr)
		b.Stderr = f
	case fdBoth:
		if err := sameFile(b.Stdin, f); err != nil {
			return err
		}
		closeFile(b.Stdout)
		closeFile(b.Stderr)
		b.Stdout, b.Stderr = f, f
	default:
		return fmt.Errorf("invalid file description %d", fd)
	}
	return nil
}

func (b *Builtin) enable(e bool) {
	b.disabled = e
}

func (b *Builtin) closeDescriptors() {
	if c, ok := b.Stdin.(io.Closer); ok {
		c.Close()
	}
	if c, ok := b.Stdout.(io.Closer); ok {
		c.Close()
	}
	if c, ok := b.Stderr.(io.Closer); ok {
		c.Close()
	}
}

var builtins map[string]Builtin

func init() {
	builtins = map[string]Builtin{
		"echo": {
			Usage: "echo [-i] [-h] [arg...]",
			Short: "write arguments to standard output",
			Exec:  Echo,
		},
		"date": {
			Usage: "date [-u] [-h]",
			Short: "write current date time to standart output",
			Exec:  Date,
		},
		"help": {
			Usage: "help [-h] builtin",
			Short: "print help text for a builtin command",
			Exec:  Help,
		},
		"builtins": {
			Usage: "builtins [-h]",
			Short: "print list of builtins and a short description",
			Exec:  Builtins,
		},
		"rand": {
			Usage: "rand [-h]",
			Short: fmt.Sprintf("generate a random integer between 0 and %d", math.MaxUint32),
			Exec:  Random,
		},
		"printf": {
			Usage: "printf [-v] [-h] format [arg...]",
			Short: "write the formatted arguments to standard output",
			Exec:  Printf,
		},
		"local": {
			Usage: "local [-h] name[=value]...",
			Short: "create a variable with name and assign a optional value",
			Exec:  Local,
		},
		"true": {
			Usage: "true [-h]",
			Short: "always returns a successfull result",
			Exec:  True,
		},
		"false": {
			Usage: "false [-h]",
			Short: "always returns a unsuccessfull result",
			Exec:  False,
		},
		"seq": {
			Usage: "seq [-h] [lower] [upper] [increment]",
			Short: "print a sequence of numbers",
			Exec:  Seq,
		},
		"type": {
			Usage: "type [-h] [-n] name [name...]",
			Short: "show information about command type",
			Exec:  Type,
		},
		"exit": {
			Usage: "exit [-h] [status]",
			Short: "exit the shell with the given status",
			Exec:  Exit,
		},
		"env": {
			Usage: "env [-h] [variable...]",
			Short: "print environment variables on stdout",
			Exec:  Environ,
		},
		"readonly": {
			Usage: "readonly [-h] ",
			Short: "",
			Exec:  Readonly,
		},
		"export": {
			Usage: "export [-h] name[=value]...",
			Short: "export variable to the shell environment",
			Exec:  Export,
		},
		"enable": {
			Usage: "enable [-n] [-f] [-r] [-h] [builtin...]",
			Short: "enable and/or disable shell builtins",
			Exec:  Enable,
		},
		"command": {
			Usage: "command name [option] [arg...]",
			Short: "execute command without performing builtin lookup",
			Exec:  ExecCommand,
		},
		"builtin": {
			Usage: "builtin name [option] [arg...]",
			Short: "execute builtin without performing command lookup",
			Exec:  ExecBuiltin,
		},
		"alias": {
			Usage: "alias name=value",
			Short: "register an alias with given name",
			Exec:  Alias,
		},
		"unalias": {
			Usage: "unalias [-a] [name...]",
			Short: "unregister an alias with given name or all",
			Exec:  Unalias,
		},
		"pwd": {
			Usage: "pwd",
			Short: "print the name of the current working directory",
			Exec:  WorkDir,
		},
		"cd": {
			Usage: "cd [dir]",
			Short: "change the shell current working directory",
			Exec:  Chdir,
		},
		"pushd": {
			Usage: "pushd [dir]",
			Short: "",
			Exec:  PushDir,
		},
		"popd": {
			Usage: "popd [dir]",
			Short: "",
			Exec:  PopDir,
		},
		"chroot": {
			Usage: "chroot dir",
			Short: "change root directory",
			Exec:  Chroot,
		},
		// "time":    {},
		// "source":  {},
	}
}

func ExecBuiltin(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	return ExitOk
}

func ExecCommand(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	return ExitOk
}

func Chroot(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	return ExitOk
}

func PushDir(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}

	if step, errc := strconv.ParseInt(set.Arg(0), 10, 64); errc == nil {
		b.PushDir(step)
	} else {
		err := b.Chdir(set.Arg(0))
		if err != nil {
			fmt.Fprintln(b.Stderr, err)
			return ExitBadUsage
		}
	}
	return ExitOk
}

func PopDir(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	step, err := strconv.ParseInt(set.Arg(0), 10, 64)
	if err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	b.PopDir(step)
	return ExitOk
}

func Chdir(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	dir := set.Arg(0)
	if dir == "" {
		vs, _ := b.Resolve("HOME")
		if len(vs) == 0 {
			fmt.Fprintln(b.stderr, "$HOME not define")
			return ExitVariable
		}
		dir = vs[0]
	}
	if err := b.Chdir(dir); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitNoFile
	}
	return ExitOk
}

func WorkDir(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	fmt.Fprintln(b.Stdout, b.Cwd())
	return ExitOk
}

func Alias(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help || set.NArg() == 0 {
		set.Usage()
		return ExitHelp
	}
	for _, a := range set.Args() {
		ix := strings.Index(a, "=")
		if ix <= 0 {
			fmt.Fprintf(b.Stderr, "%s: missing equal or alias name\n", a)
			continue
		}
		if err := b.RegisterAlias(a[:ix], a[ix+1:]); err != nil {
			fmt.Fprintln(b.Stderr, err)
			return ExitUnknown
		}
	}

	return ExitOk
}

func Unalias(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		all  = set.Bool("a", false, "remove all registered aliases")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	if *all {
		b.UnregisterAlias("")
	}
	for _, a := range set.Args() {
		b.UnregisterAlias(a)
	}
	return ExitOk
}

func Printf(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		name = set.String("v", "", "assign output to variable")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	args := set.Args()
	if len(args) < 2 {
		return ExitBadUsage
	}

	vs := make([]interface{}, len(args)-1)
	for i := 0; i < len(vs); i++ {
		vs[i] = args[i+1]
	}

	str := fmt.Sprintf(args[0], vs...)
	if *name != "" {
		err := b.Define(*name, []string{str})
		if err != nil {
			fmt.Fprintln(b.Stderr, err)
			return ExitVariable
		}
		return ExitOk
	}
	fmt.Fprintln(b.Stdout, str)
	return ExitOk
}

func Random(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		seed = set.Int64("s", time.Now().Unix(), "use SEED to seed the generator")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	rand.Seed(*seed)
	fmt.Fprintf(b.Stdout, "%d\n", rand.Uint32())
	return ExitOk
}

func Echo(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		in   = set.Bool("i", false, "read arguments from stdin")
		nonl = set.Bool("n", false, "do not append newline at end of line")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}

	if !*in {
		fmt.Fprint(b.Stdout, strings.Join(set.Args(), " "))
		if !*nonl {
			fmt.Fprintln(b.Stdout)
		}
		return ExitOk
	}
	s := bufio.NewScanner(b.Stdin)
	for s.Scan() {
		fmt.Fprint(b.Stdout, s.Text())
		if !*nonl {
			fmt.Fprintln(b.Stdout)
		}
	}
	return ExitOk
}

func Date(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		utc  = set.Bool("u", false, "utc time")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	var delta time.Duration
	switch strings.ToLower(set.Arg(0)) {
	case "yesterday":
		delta = -24 * time.Hour
	case "tomorrow":
		delta = 24 * time.Hour
	default:
	}
	now := time.Now().Add(delta)
	if *utc {
		now = now.UTC()
	}
	fmt.Fprintln(b.Stdout, now.Format("2006-01-02 15:04:05"))
	return ExitOk
}

func Builtins(b Builtin) ErrCode {
	var (
		set    = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		intern = set.Bool("i", false, "only show default builtins")
		help   = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	for k, c := range builtins {
		if !c.Runnable() {
			continue
		}
		if c.external && *intern {
			continue
		}
		fmt.Printf("%s: %s\n", k, c.Short)
	}
	return ExitOk
}

func Help(b Builtin) ErrCode {
	var (
		set    = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		intern = set.Bool("i", false, "only show default builtins")
		short  = set.Bool("s", false, "only show short help message")
		help   = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	if set.NArg() == 0 {
		fmt.Fprintln(b.Stderr, b.Short)
		return ExitBadUsage
	}
	x, ok := builtins[set.Arg(0)]
	if !ok || (x.external && *intern) {
		fmt.Fprintf(b.Stderr, "%s: unknown builtin\n", set.Arg(0))
		return ExitUnknown
	}
	fmt.Fprintln(b.Stdout, x.String())
	fmt.Fprintln(b.Stdout, x.Short)
	if !*short {
		if x.Desc != "" {
			fmt.Fprintln(b.Stdout)
			fmt.Fprintln(b.Stdout, x.Desc)
		}
		fmt.Fprintln(b.Stdout)
		fmt.Fprintln(b.Stdout, "usage:", x.Usage)
	}
	return ExitOk
}

func True(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	return ExitOk
}

func False(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	return ExitVariable
}

func Seq(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		pat  = set.String("f", "%d", "format output number")
		sep  = set.String("s", "\n", "separate number with string")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	var (
		lower int64
		upper int64
		incr  int64
		err   error
		cmp   = func(lower, upper int64) bool { return lower <= upper }
	)
	switch set.NArg() {
	case 0:
		fmt.Fprintln(b.Stderr, "not enough arguments given")
		return ExitBadUsage
	case 1:
		x, err := strconv.ParseInt(set.Arg(0), 10, 64)
		if err != nil {
			fmt.Fprintln(b.Stderr, err)
			return ExitBadUsage
		}
		if x > 0 {
			upper = x
		} else if x < 0 {
			lower = x
		} else {
			return ExitOk
		}
		incr++
	case 2:
		var err error
		if lower, err = strconv.ParseInt(set.Arg(0), 10, 64); err != nil {
			break
		}
		if upper, err = strconv.ParseInt(set.Arg(1), 10, 64); err != nil {
			break
		}
		if lower < upper {
			incr++
		} else {
			incr--
		}

		if lower < 0 && upper < 0 && lower > upper {
			cmp = func(lower, upper int64) bool { return lower >= upper }
		}
	case 3:
		var err error
		if lower, err = strconv.ParseInt(set.Arg(0), 10, 64); err != nil {
			break
		}
		if upper, err = strconv.ParseInt(set.Arg(1), 10, 64); err != nil {
			break
		}
		if incr, err = strconv.ParseInt(set.Arg(2), 10, 64); err != nil {
			break
		}
		if x := lower + incr; x < lower {
			return ExitOk
		}
	default:
		fmt.Fprintln(b.Stderr, "too many arguments given")
		return ExitBadUsage
	}
	if err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	var str []string
	for cmp(lower, upper) {
		str = append(str, fmt.Sprintf(*pat, lower))
		lower += incr
	}
	fmt.Fprintln(b.Stdout, strings.Join(str, *sep))
	return ExitOk
}

func Type(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		nob  = set.Bool("n", false, "exclude builtin")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	for _, a := range set.Args() {
		if _, ok := builtins[a]; ok && !*nob {
			fmt.Fprintf(b.Stdout, "%s: builtin\n", a)
			continue
		}
		if _, ok := b.alias[a]; ok {
			fmt.Fprintf(b.Stdout, "%s: alias\n", a)
			continue
		}
		// will look later for functions - when builtin will have access to it
		if _, err := exec.LookPath(a); err == nil {
			fmt.Fprintf(b.Stdout, "%s: command\n", a)
			continue
		}
		i, err := os.Stat(a)
		if err != nil {
			fmt.Fprintf(b.Stderr, "%s: no such file or directory\n", a)
			continue
		}
		if i.Mode().IsRegular() {
			fmt.Fprintf(b.Stdout, "%s: file\n", a)
			continue
		}
		switch m := i.Mode(); {
		case m&os.ModeDir == os.ModeDir:
			fmt.Fprintf(b.Stdout, "%s: directory\n", a)
		case m&os.ModeSymlink == os.ModeSymlink:
			fmt.Fprintf(b.Stdout, "%s: symlink\n", a)
		case m&os.ModeSocket == os.ModeSocket:
			fmt.Fprintf(b.Stdout, "%s: socket\n", a)
		case m&os.ModeNamedPipe == os.ModeNamedPipe:
			fmt.Fprintf(b.Stdout, "%s: pipe\n", a)
		default:
			fmt.Fprintf(b.Stderr, "%s: unknown\n", a)
		}
	}
	return ExitOk
}

func Exit(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	if set.NArg() == 0 {
		return ExitOk
	}
	exit, err := strconv.Atoi(set.Arg(0))
	if err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	b.Exit(ErrCode(exit))
	return ExitOk
}

func Environ(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}

	es := make([]string, 0, set.NArg())
	for _, e := range set.Args() {
		vs, _ := b.Resolve(e)
		if len(vs) == 0 {
			continue
		}
		es = append(es, fmt.Sprintf("%s=%s", e, strings.Join(vs, " ")))
	}
	if len(es) == 0 {
		es = b.Environ()
	}
	for _, e := range es {
		fmt.Fprintln(b.Stdout, e)
	}
	return ExitOk
}

func Local(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	for _, a := range set.Args() {
		var (
			opt string
			val string
			ix  = strings.IndexByte(a, '=')
		)
		if ix > 0 {
			opt, val = a[:ix], a[ix+1:]
		} else if ix < 0 {
			opt = a
		} else {
			fmt.Fprintf(b.Stderr, "%s: missing variable name\n", a)
		}
		b.Define(opt, []string{val})
	}
	return ExitOk
}

func Export(b Builtin) ErrCode {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}

	for _, a := range set.Args() {
		var (
			opt string
			val string
			ix  = strings.IndexByte(a, '=')
		)
		if ix > 0 {
			opt, val = a[:ix], a[ix+1:]
		} else if ix < 0 {
			opt = a
		} else {
			fmt.Fprintf(b.Stderr, "%s: missing variable name\n", a)
		}
		b.Export(opt, []string{val})
	}
	return ExitOk
}

func Readonly(b Builtin) ErrCode {
	var (
		set   = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		ronly = set.Bool("n", false, "")
		help  = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help {
		set.Usage()
		return ExitHelp
	}
	for _, a := range set.Args() {
		b.SetReadOnly(a, *ronly)
	}
	return ExitOk
}

func Enable(b Builtin) ErrCode {
	var (
		set       = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		file      = set.Bool("f", false, "register new builtins from external plugins")
		disabled  = set.Bool("n", false, "disabled the builtins")
		overwrite = set.Bool("r", false, "replace builtin")
		help      = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.Stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		fmt.Fprintln(b.Stderr, err)
		return ExitBadUsage
	}
	if *help || set.NArg() == 0 {
		set.Usage()
		return ExitHelp
	}
	if !*file {
		for _, a := range set.Args() {
			b, ok := builtins[a]
			if !ok {
				fmt.Fprintf(b.Stderr, "%s: builtin not found", a)
				continue
			}
			b.enable(*disabled)
		}
		return ExitOk
	}
	for _, a := range set.Args() {
		p, err := plugin.Open(a)
		if err != nil {
			fmt.Fprintln(b.Stderr, err)
			return ExitIO
		}
		sym, err := p.Lookup("Builtins")
		if err != nil {
			fmt.Fprintln(b.Stderr, err)
			return ExitUnknown
		}
		list, ok := sym.(func() []*Builtin)
		if !ok {
			fmt.Fprintln(b.Stderr, "invalid signature")
		}
		for _, b := range list() {
			if _, ok := builtins[b.String()]; ok && !*overwrite {
				continue
			}
			b.external = true
			builtins[b.String()] = *b
		}
	}
	return ExitOk
}
