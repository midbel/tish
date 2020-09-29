package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/midbel/tish"
)

type PrintFunc func(tish.Command, tish.Environment)

func main() {
	ast := flag.Bool("a", false, "print AST")
	flag.Parse()
	var print PrintFunc
	if *ast {
		print = func(cmd tish.Command, _ tish.Environment) {
			fmt.Fprintln(os.Stdout, cmd)
		}
	} else {
		print = func(cmd tish.Command, env tish.Environment) {

		}
	}
	for _, a := range flag.Args() {
		if err := PrintCommand(strings.NewReader(a), print); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func PrintCommand(r io.Reader, print PrintFunc) error {
	p, err := tish.NewParser(r)
	if err != nil {
		return err
	}
	env := tish.Environ()
	for {
		cmd, err := p.Parse()
		switch err {
		default:
			return err
		case io.EOF:
			return nil
		case nil:
			print(cmd, env)
		}
	}
}
