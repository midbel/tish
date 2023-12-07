package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	limit := flag.Int64("n", 0, "number of bytes")
	flag.Parse()

	for _, f := range flag.Args() {
		if err := cat(f, *limit); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func cat(file string, limit int64) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	var rs io.Reader = r
	if limit > 0 {
		rs = io.LimitReader(r, limit)
	}
	_, err = io.Copy(os.Stdout, rs)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return err
}
