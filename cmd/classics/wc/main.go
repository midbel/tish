package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Parse()

	var results []counter
	for _, f := range flag.Args() {
		c, err := countLine(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		results = append(results, c)
	}
	for i, f := range flag.Args() {
		fmt.Fprintf(os.Stdout, "%s %d %d %d", f, results[i].Lines, results[i].Words, results[i].Chars)
		fmt.Fprintln(os.Stdout)
	}
}

type counter struct {
	Lines int
	Words int
	Chars int
}

func countLine(file string) (counter, error) {
	r, err := os.Open(file)
	if err != nil {
		return counter{}, err
	}
	defer r.Close()

	var (
		scan = bufio.NewScanner(r)
		res  counter
	)
	for scan.Scan() {
		res.Lines++
		res.Words += countWords(scan.Bytes())
		res.Chars += len(scan.Bytes())
	}
	return res, scan.Err()
}

func countWords(line []byte) int {
	scan := bufio.NewScanner(bytes.NewReader(line))
	scan.Split(bufio.ScanWords)

	var res int
	for scan.Scan() {
		res++
	}
	return res
}
