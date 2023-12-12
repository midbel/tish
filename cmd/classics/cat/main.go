package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	var (
		limit = flag.Int64("c", 0, "number of bytes")
		number = flag.Bool("n", false, "print line number")
	)
	flag.Parse()

	for _, f := range flag.Args() {
		if err := cat(f, *limit, *number); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func cat(file string, limit int64, number bool) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	var rs io.Reader = r
	if limit > 0 {
		rs = io.LimitReader(r, limit)
	}
	if number {
		return printNumber(rs)
	}
	_, err = io.Copy(os.Stdout, rs)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return err
}

func printNumber(r io.Reader) error {
	scan := bufio.NewScanner(r)
	for i := 1; scan.Scan(); i++ {
		fmt.Printf("%4d%s", i, scan.Text())
		fmt.Println()
	}
	return scan.Err()
}
