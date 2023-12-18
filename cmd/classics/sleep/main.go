package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	flag.Parse()
	var (
		sec time.Duration
		err error
	)
	if sec, err = time.ParseDuration(flag.Arg(0)); err != nil {
		n, err := strconv.Atoi(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, "fail to parse argument as number")
			os.Exit(1)
		}
		sec = time.Duration(n) * time.Second
	}

	time.Sleep(sec)
}
