package main

import (
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
	if err := scanFile(r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func scanFile(r io.Reader) error {
	scan, err := tish.Scan(r)
	if err != nil {
		return err
	}
	for {
		tok := scan.Scan()
		if tok.Type == tish.EOF {
			break
		}
		if tok.Type == tish.Invalid {
			return fmt.Errorf("invalid token: ", tok.String())
		}
		fmt.Println(tok)
	}
	return nil
}
