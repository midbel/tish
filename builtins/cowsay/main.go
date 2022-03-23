package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/midbel/tish"
)

const cow = `
^__^
(oo)\_______
(__)\       )\/\
    ||----w |
    ||     ||
`

func Load() tish.Builtin {
	return tish.Builtin{
		Usage:   "cowsay <message>",
		Short:   "set specific shell option",
		Help:    "",
		Execute: runCowsay,
	}
}

func runCowsay(b tish.Builtin) error {
	var set flag.FlagSet
	if err := set.Parse(b.Args); err != nil {
		return err
	}
	fmt.Fprintln(b.Stdout, strings.TrimSpace(cow))
	return nil
}
