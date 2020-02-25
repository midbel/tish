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

type Command interface {
	Start() error
	Wait() error
	Run() error
}

type Builtin struct {
	Usage string
	Short string
	Desc  string

	disabled bool
	external bool

	Args []string
	Env  *Env
	Exec func(Builtin) error

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	finished bool
	done     chan error
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
		return fmt.Errorf("%s: not runnable", b.String())
	}
	if b.finished {
		return fmt.Errorf("%s: already executed", b.String())
	}

	b.done = make(chan error, 1)
	go func() {
		b.done <- b.Exec(*b)
		b.closeStreams()
		close(b.done)
	}()
	return nil
}

func (b *Builtin) Wait() error {
	if !b.Runnable() {
		return fmt.Errorf("%s: not runnable", b.String())
	}
	if b.finished {
		return fmt.Errorf("%s: already executed", b.String())
	}
	b.finished = true

	return <-b.done
}

func (b *Builtin) Run() error {
	if err := b.Start(); err != nil {
		return err
	}
	return b.Wait()
}

func (b *Builtin) enable(e bool) {
	b.disabled = e
}

func (b *Builtin) closeStreams() {
	if c, ok := b.stdin.(io.Closer); ok {
		c.Close()
	}
	if c, ok := b.stdout.(io.Closer); ok {
		c.Close()
	}
	if c, ok := b.stdout.(io.Closer); ok {
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
			Short: "",
			Exec:  Export,
		},
		"enable": {
			Usage: "enable [-n] [-f] [-r] [-h] [builtin...]",
			Short: "",
			Exec:  Enable,
		},
		"command": {
			Usage: "command name [option] arguments...",
			Short: "",
			Exec:  nil,
		},
		"builtin": {
			Usage: "builtin name [option] arguments...",
			Short: "",
			Exec:  nil,
		},
		// "alias":   {},
		// "unalias": {},
		// "pwd":     {},
		// "cd":      {},
		// "time":    {},
		// "source":  {},
	}
}

func Printf(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		name = set.String("v", "", "variable")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	args := set.Args()
	if len(args) < 2 {
		return nil
	}
	vs := make([]interface{}, len(args)-1)
	for i := 0; i < len(vs); i++ {
		vs[i] = args[i+1]
	}
	str := fmt.Sprintf(args[0], vs...)
	if *name != "" {
		return nil
	}
	_, err := fmt.Fprintln(b.stdout, str)
	return err
}

func Random(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		seed = set.Int64("s", time.Now().Unix(), "use SEED to seed the generator")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	rand.Seed(*seed)
	_, err := fmt.Fprintf(b.stdout, "%d\n", rand.Uint32())
	return err

}

func Echo(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		in   = set.Bool("i", false, "read arguments from stdin")
		nonl = set.Bool("n", false, "do not append newline at end of line")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}

	if !*in {
		_, err := fmt.Fprint(b.stdout, strings.Join(set.Args(), " "))
		if !*nonl {
			_, err = fmt.Fprintln(b.stdout)
		}
		return err
	}
	s := bufio.NewScanner(b.stdin)
	for s.Scan() {
		_, err := fmt.Fprint(b.stdout, s.Text())
		if !*nonl {
			_, err = fmt.Fprintln(b.stdout)
		}
		if err != nil {
			return err
		}
	}
	return s.Err()
}

func Date(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		utc  = set.Bool("u", false, "utc time")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	now := time.Now()
	if *utc {
		now = now.UTC()
	}
	_, err := fmt.Fprintln(b.stdout, now.Format("2006-01-02 15:04:05"))
	return err
}

func Builtins(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	for k, c := range builtins {
		if !c.Runnable() {
			continue
		}
		fmt.Printf("%s: %s\n", k, c.Short)
	}
	return nil
}

func Help(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	if set.NArg() == 0 {
		return nil
	}
	x, ok := builtins[set.Arg(0)]
	if !ok {
		return fmt.Errorf("unknown builtin: %s", set.Arg(0))
	}
	fmt.Fprintln(b.stdout, x.String())
	fmt.Fprintln(b.stdout, x.Short)
	if x.Desc != "" {
		fmt.Fprintln(b.stdout)
		fmt.Fprintln(b.stdout, x.Desc)
	}
	fmt.Fprintln(b.stdout)
	fmt.Fprintln(b.stdout, "usage:", x.Usage)
	return nil
}

func True(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	return nil
}

func False(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	return fmt.Errorf(b.String())
}

func Seq(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		pat  = set.String("f", "%d", "format output number")
		sep  = set.String("s", "\n", "separate number with string")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
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
		err = fmt.Errorf("not enough arguments given")
	case 1:
		x, err := strconv.ParseInt(set.Arg(0), 10, 64)
		if err != nil {
			return err
		}
		if x > 0 {
			upper = x
		} else if x < 0 {
			lower = x
		} else {
			return nil
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
			return nil
		}
	default:
		err = fmt.Errorf("too many arguments given")
	}
	if err != nil {
		return err
	}
	var str []string
	for cmp(lower, upper) {
		str = append(str, fmt.Sprintf(*pat, lower))
		lower += incr
	}
	fmt.Fprintln(b.stdout, strings.Join(str, *sep))
	return nil
}

func Type(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		nob  = set.Bool("n", false, "exclude builtin")
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	for _, a := range set.Args() {
		if _, ok := builtins[a]; ok && !*nob {
			fmt.Fprintf(b.stdout, "%s: builtin\n", a)
			continue
		}
		// will look later for alias and/or functions - when builtin will have access to it
		if _, err := exec.LookPath(a); err == nil {
			fmt.Fprintf(b.stdout, "%s: command\n", a)
			continue
		}
		i, err := os.Stat(a)
		if err != nil {
			fmt.Fprintf(b.stderr, "%s: no such file or directory\n", a)
			continue
		}
		if i.Mode().IsRegular() {
			fmt.Fprintf(b.stdout, "%s: file\n", a)
			continue
		}
		switch m := i.Mode(); {
		case m&os.ModeDir == os.ModeDir:
			fmt.Fprintf(b.stdout, "%s: directory\n", a)
		case m&os.ModeSymlink == os.ModeSymlink:
			fmt.Fprintf(b.stdout, "%s: symlink\n", a)
		case m&os.ModeSocket == os.ModeSocket:
			fmt.Fprintf(b.stdout, "%s: socket\n", a)
		case m&os.ModeNamedPipe == os.ModeNamedPipe:
			fmt.Fprintf(b.stdout, "%s: pipe\n", a)
		default:
			fmt.Fprintf(b.stderr, "%s: unknown\n", a)
		}
	}
	return nil
}

func Exit(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	if set.NArg() == 0 {
		return nil
	}
	_, err := strconv.Atoi(set.Arg(0))
	if err != nil {
		return err
	}
	return nil
}

func Environ(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}

	es := make([]string, 0, set.NArg())
	for _, e := range set.Args() {
		vs, err := b.Env.Get(e)
		if err != nil {
			continue
		}
		es = append(es, fmt.Sprintf("%s=%s", e, strings.Join(vs, " ")))
	}
	if len(es) == 0 {
		es = b.Env.Values()
	}
	for _, e := range es {
		fmt.Fprintln(b.stdout, e)
	}
	return nil
}

func Local(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	env, err := b.Env.Unwrap()
	if err != nil {
		return err
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
			fmt.Fprintf(b.stderr, "%s: missing variable name\n", a)
		}
		env.Set(opt, []string{val})
	}
	return nil
}

func Export(b Builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		help = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	env := b.Env
	for {
		e, err := env.Unwrap()
		if err != nil {
			env = e
			break
		}
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
			fmt.Fprintf(b.stderr, "%s: missing variable name\n", a)
		}
		env.Set(opt, []string{val})
	}
	return nil
}

func Readonly(b Builtin) error {
	var (
		set   = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		ronly = set.Bool("n", false, "")
		help  = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help {
		set.Usage()
		return nil
	}
	for _, a := range set.Args() {
		b.Env.SetReadOnly(a, *ronly)
	}
	return nil
}

func Enable(b Builtin) error {
	var (
		set       = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		file      = set.Bool("f", false, "register new builtins from external plugins")
		disabled  = set.Bool("n", false, "disabled the builtins")
		overwrite = set.Bool("r", false, "replace builtin")
		help      = set.Bool("h", false, "show help message and exit")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	if *help || set.NArg() == 0 {
		set.Usage()
		return nil
	}
	if !*file {
		for _, a := range set.Args() {
			b, ok := builtins[a]
			if !ok {
				fmt.Fprintf(b.stderr, "%s: builtin not found", a)
				continue
			}
			b.enable(*disabled)
		}
		return nil
	}
	for _, a := range set.Args() {
		p, err := plugin.Open(a)
		if err != nil {
			return err
		}
		sym, err := p.Lookup("Builtins")
		if err != nil {
			return err
		}
		list, ok := sym.(func() []*Builtin)
		if !ok {
			return fmt.Errorf("invalid signature")
		}
		for _, b := range list() {
			if _, ok := builtins[b.String()]; ok && !*overwrite {
				continue
			}
			b.external = true
			builtins[b.String()] = *b
		}
	}
	return nil
}
