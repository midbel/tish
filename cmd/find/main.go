package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var (
	withAttr bool
	withFile bool
)

func main() {
	flag.BoolVar(&withAttr, "l", withAttr, "print file propery")
	flag.BoolVar(&withFile, "f", withFile, "keep only regular file")
	flag.Parse()
	files, err := filepath.Glob(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, f := range files {
		i, err := os.Stat(f)
		if err != nil || (!withFile && !i.Mode().IsRegular()) {
			continue
		}
		if !withAttr {
			fmt.Fprintln(os.Stdout, f)
		} else {
			when := i.ModTime()
			size := i.Size()
			mode := i.Mode()
			fmt.Fprintf(os.Stdout, "%s %s %8d %s\n", mode, when.Format("2006-01-02"), size, f)
		}
	}
}
