package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/midbel/tish"
)

func main() {
	flag.Parse()

	var r io.Reader
	if f, err := os.Open(flag.Arg(0)); err == nil {
		defer f.Close()
		r = f
	} else {
		r = strings.NewReader(flag.Arg(0))
		if flag.NArg() == 0 {
			r = os.Stdin
		}
	}
	if err := parseFile(r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseFile(r io.Reader) error {
	p, err := tish.New(r)
	if err != nil {
		return err
	}
	for {
		cmd, err := p.Parse()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "%#v\n", cmd)
	}
	return nil
}
