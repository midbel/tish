package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/tish"
)

func main() {
	flag.Parse()

	err := tish.Dump(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
