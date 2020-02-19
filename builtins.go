package tish

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"strings"
	"time"
)

type Command interface {
	Start() error
	Wait() error
	Run() error
}

type builtin struct {
	Usage string
	Short string
	Desc  string

	args []string
	run  func(builtin) error

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	finished bool
	done     chan error
}

func (b *builtin) String() string {
	if i := strings.Index(b.Usage, " "); i > 0 {
		return b.Usage[:i]
	}
	return b.Usage
}

func (b *builtin) Help() string {
	var buf strings.Builder
	buf.WriteString(b.Short)
	if b.Desc != "" {
		buf.WriteRune(newline)
		buf.WriteString(b.Desc)
	}
	buf.WriteRune(newline)
	buf.WriteString(b.Usage)
	return buf.String()
}

func (b *builtin) Runnable() bool {
	return b.run != nil
}

func (b *builtin) Start() error {
	if !b.Runnable() {
		return fmt.Errorf("%s: not runnable", b.String())
	}
	if b.finished {
		return fmt.Errorf("%s: already executed", b.String())
	}

	b.done = make(chan error, 1)
	go func() {
		b.done <- b.run(*b)
		b.closeStreams()
		close(b.done)
	}()
	return nil
}

func (b *builtin) Wait() error {
	if !b.Runnable() {
		return fmt.Errorf("%s: not runnable", b.String())
	}
	if b.finished {
		return fmt.Errorf("%s: already done", b.String())
	}
	b.finished = true

	return <-b.done
}

func (b *builtin) Run() error {
	if err := b.Start(); err != nil {
		return err
	}
	return b.Wait()
}

func (b *builtin) closeStreams() {
	if c, ok := b.stdin.(io.Closer); ok && b.stdin != stdin {
		c.Close()
	}
	if c, ok := b.stdout.(io.Closer); ok && b.stdout != stdout {
		c.Close()
	}
	if c, ok := b.stdout.(io.Closer); ok && b.stderr != stderr {
		c.Close()
	}
}

var builtins map[string]builtin

func init() {
	builtins = map[string]builtin{
		"echo": {
			Usage: "echo [arg...]",
			Short: "write arguments to standard output",
			run:   Echo,
		},
		"date": {
			Usage: "date [-u]",
			Short: "write current date time to standart output",
			run:   Date,
		},
		"help": {
			Usage: "help builtin",
			Short: "print help text for a builtin command",
			run:   Help,
		},
		"builtins": {
			Usage: "builtins",
			Short: "print list of builtins and a short description",
			run:   Builtins,
		},
		"rand": {
			Usage: "rand",
			Short: fmt.Sprintf("generate a random integer between 0 and %d", math.MaxUint32),
			run:   Random,
		},
		"printf": {
			Usage: "printf [-v var] format [arg...]",
			Short: "write the formatted arguments to standard output",
			run:   Printf,
		},
		"local": {
			Usage: "local name[=value]...",
			Short: "create a variable with name and assign a optional value",
			run:   Local,
		},
		"true": {
			Usage: "true",
			Short: "always returns a successfull result",
			run:   True,
		},
		"false": {
			Usage: "false",
			Short: "always returns a unsuccessfull result",
			run:   False,
		}
		// "env":     {},
		// "export":  {},
		// "alias":   {},
		// "unalias": {},
		// "pwd":     {},
		// "cd":      {},
		// "pwd":     {},
		// "time":    {},
		// "type":    {},
	}
}

func Local(b builtin) error {
	set := flag.NewFlagSet(b.String(), flag.ContinueOnError)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.args); err != nil {
		return err
	}
	return nil
}

func Printf(b builtin) error {
	var (
		set  = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		name = set.String("v", "", "variable")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.args); err != nil {
		return err
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

func Random(b builtin) error {
	set := flag.NewFlagSet(b.String(), flag.ContinueOnError)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.args); err != nil {
		return err
	}
	_, err := fmt.Fprintf(b.stdout, "%d\n", rand.Uint32())
	return err

}

func Echo(b builtin) error {
	var (
		set = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		in  = set.Bool("i", false, "read arguments from stdin")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.args); err != nil {
		return err
	}
	if !*in {
		_, err := fmt.Fprintln(b.stdout, strings.Join(set.Args(), " "))
		return err
	}
	s := bufio.NewScanner(b.stdin)
	for s.Scan() {
		_, err := fmt.Fprintln(b.stdout, s.Text())
		if err != nil {
			return err
		}
	}
	return s.Err()
}

func Date(b builtin) error {
	var (
		set = flag.NewFlagSet(b.String(), flag.ContinueOnError)
		utc = set.Bool("u", false, "utc time")
	)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.args); err != nil {
		return err
	}
	now := time.Now()
	if *utc {
		now = now.UTC()
	}
	_, err := fmt.Fprintln(b.stdout, now.Format("2006-01-02 15:04:05"))
	return err
}

func Builtins(b builtin) error {
	set := flag.NewFlagSet(b.String(), flag.ContinueOnError)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.args); err != nil {
		return err
	}
	for k, c := range builtins {
		if !c.Runnable() {
			continue
		}
		fmt.Printf("%s: %s\n", k, c.Short)
	}
	return nil
}

func Help(b builtin) error {
	set := flag.NewFlagSet(b.String(), flag.ContinueOnError)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.args); err != nil {
		return err
	}
	if set.NArg() == 0 {
		return nil
	}
	x, ok := builtins[set.Arg(0)]
	if !ok {
		return fmt.Errorf("unknown builtin: %s", flag.Arg(0))
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

func True(b builtin) error {
	set := flag.NewFlagSet(b.String(), flag.ContinueOnError)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.args); err != nil {
		return err
	}
	return nil
}

func False(b builtin) error {
	set := flag.NewFlagSet(b.String(), flag.ContinueOnError)
	set.Usage = func() {
		fmt.Fprintln(b.stderr, b.Help())
	}
	if err := set.Parse(b.args); err != nil {
		return err
	}
	return fmt.Errorf(b.String())
}
