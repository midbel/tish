package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

var (
	withBlank  bool
	withNumber bool
	withEnding bool
)

func main() {
	flag.BoolVar(&withBlank, "b", withBlank, "")
	flag.BoolVar(&withNumber, "n", withNumber, "")
	flag.BoolVar(&withEnding, "e", withEnding, "")
	flag.Parse()
	if flag.NArg() == 0 {
		if err := copyFile(os.Stdin); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	var exit int
	for _, a := range flag.Args() {
		if err := openFile(a); err != nil {
			fmt.Fprintln(os.Stderr, err)
			exit++
		}
	}
	os.Exit(exit)
}

func openFile(file string) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()
	return copyFile(r)
}

func copyFile(r io.Reader) error {
	s := bufio.NewScanner(r)
	for i := 1; s.Scan(); i++ {
		line := s.Text()
		if line == "" && !withBlank {
			continue
		}
		switch {
		case withNumber && !withEnding:
			fmt.Fprintf(os.Stdout, "%4d: %s", i, line)
		case !withNumber && withEnding:
			fmt.Fprintf(os.Stdout, "%s$", line)
		case withNumber && withEnding:
			fmt.Fprintf(os.Stdout, "%4d: %s$", i, line)
		default:
			fmt.Fprint(os.Stdout, line)
		}
		fmt.Fprintln(os.Stdout)
	}
	return s.Err()
}
