package tish

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

type Command struct {
	Usage string
	Short string
	Desc  string
	Run   func(Command, []string) error
}

func (c Command) String() string {
	ps := strings.Split(c.Usage, " ")
	return ps[0]
}

func (c Command) Runnable() bool {
	return c.Run != nil
}

var builtins map[string]Command

func init() {
	builtins = map[string]Command{
		"echo": {
			Usage: "echo [arg...]",
			Short: "write arguments to standard output",
			Run:   Echo,
		},
		"date": {
			Usage: "date [-u]",
			Short: "write current date time to standart output",
			Run:   Date,
		},
		"help": {
			Usage: "help builtin",
			Short: "print help text for a builtin command",
			Run:   Help,
		},
		"builtins": {
			Usage: "builtins",
			Short: "print list of builtins and a short description",
			Run:   Builtins,
		},
		"rand": {
			Usage: "rand",
			Short: fmt.Sprintf("generate a random integer between 0 and %d", math.MaxUint32),
			Run:   Random,
		},
		"printf": {
			Usage: "printf [-v var] format [arg...]",
			Short: "write the formatted arguments to standard output",
			Run:   Printf,
		},
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

func Printf(c Command, args []string) error {
	var (
		set  = flag.NewFlagSet(c.String(), flag.ContinueOnError)
		name = set.String("v", "", "variable")
	)
	if err := set.Parse(args); err != nil {
		return err
	}
	args := flag.Args()
	if len(args) < 2 {
		return nil
	}
	str := fmt.Sprintf(args[0], args[1:]...)
	if *name != "" {
		return nil
	}
	_, err := fmt.Println(str)
	return err
}

func Random(c Command, args []string) error {
	set := flag.NewFlagSet(c.String(), flag.ContinueOnError)
	if err := set.Parse(args); err != nil {
		return err
	}
	_, err := fmt.Printf("%d\n", rand.Uint32())
	return err

}

func Echo(c Command, args []string) error {
	set := flag.NewFlagSet(c.String(), flag.ContinueOnError)
	if err := set.Parse(args); err != nil {
		return err
	}
	_, err := fmt.Println(strings.Join(set.Args(), " "))
	return err
}

func Date(c Command, args []string) error {
	var (
		set = flag.NewFlagSet(c.String(), flag.ContinueOnError)
		utc = set.Bool("u", false, "utc time")
	)
	if err := set.Parse(args); err != nil {
		return err
	}
	now := time.Now()
	if *utc {
		now = now.UTC()
	}
	_, err := fmt.Println(now.Format("2006-01-02 15:04:05"))
	return err
}

func Builtins(c Command, args []string) error {
	set := flag.NewFlagSet("builtins", flag.ContinueOnError)
	if err := set.Parse(args); err != nil {
		return err
	}
	for k, c := range builtins {
		if c.Run == nil {
			continue
		}
		fmt.Printf("%s: %s\n", k, c.Short)
	}
	return nil
}

func Help(c Command, args []string) error {
	set := flag.NewFlagSet("builtins", flag.ContinueOnError)
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() == 0 {
		return nil
	}
	x, ok := builtins[set.Arg(0)]
	if !ok {
		return fmt.Errorf("unknown builtin: %s", flag.Arg(0))
	}
	fmt.Println(x.String())
	fmt.Println(x.Short)
	if x.Desc != "" {
		fmt.Println()
		fmt.Println(x.Desc)
	}
	fmt.Println()
	fmt.Println("usage:", x.Usage)
	return nil
}
