package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	var (
		sep = flag.String("s", " ", "separator")
		err = flag.Bool("e", false, "write to stderr")
	)
	flag.Parse()

	var w io.Writer = os.Stdout
	if *err {
		w = os.Stderr
	}
	for i, a := range flag.Args() {
		if i > 0 {
			fmt.Fprint(w, *sep)
		}
		fmt.Fprint(w, a)
	}
	fmt.Fprintln(w)
}
