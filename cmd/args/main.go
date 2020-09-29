package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		multi = flag.Bool("m", false, "print args on multiple lines")
		quote = flag.Bool("q", false, "print args with quotes")
	)
	flag.Parse()
	pattern := "%d: %s"
	if *quote {
		pattern = "%d: %q"
	}
	if !*multi {
		fmt.Fprintf(os.Stdout, pattern, flag.NArg(), flag.Args())
		fmt.Fprintln(os.Stdout)
		return
	}
	for i, a := range flag.Args() {
		fmt.Fprintf(os.Stdout, pattern, i+1, a)
		fmt.Fprintln(os.Stdout)
	}
}
