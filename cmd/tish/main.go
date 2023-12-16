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
	// var (
	// 	verbose  = flag.Bool("v", false, "verbose")
	// 	restrict = flag.Bool("r", false, "restricted")
	// 	skiprc   = flag.Bool("norc", false, "skip rc file")
	// )
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
	if err := execFile(r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execFile(r io.Reader) error {
	sh, err := tish.NewShellWithEnv(r, tish.EmptyEnv())
	if err != nil {
		return err
	}
	sh.SetExts([]string{".exe", ".sh"})
	return sh.Run()
}
