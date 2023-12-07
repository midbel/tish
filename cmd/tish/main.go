package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/midbel/tish"
)

func main() {
	path := flag.String("d", os.Getenv("PATH"), "base path")
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
	if err := execFile(r, *path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execFile(r io.Reader, dir string) error {
	sh, err := tish.NewShellWithEnv(r, tish.EmptyEnv())
	if err != nil {
		return err
	}
	sh.SetDirs(filepath.SplitList(dir))
	sh.SetExts(".exe", ".sh")
	return sh.Run()
}
