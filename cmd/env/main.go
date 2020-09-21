package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	withEmpty bool
	withRem   bool
	withDir   string
)

func main() {
	flag.StringVar(&withDir, "c", withDir, "change working directory")
	flag.BoolVar(&withRem, "u", withRem, "remove variable from environment")
	flag.BoolVar(&withEnv, "e", withEnv, "start command with an empty environment")
	flag.Parse()
	if flag.NArg() == 0 {
		printEnv()
		return
	}
	if err := execute(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute(args []string) error {
	env, rest := splitArgs(args)
	for _, e := range env {
		kv := strings.Split(e, "=")
		os.SetEnv(kv[0], kv[1])
	}
	if len(rest) == 0 {
		return nil
	}
	cmd := rest[0]
	if len(rest) > 1 {
		rest = rest[1:]
	}
	c := exec.NewCommand(cmd, rest...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Dir = withDir
	if withEmpty {
		c.Env = []string{}
	}
	return c.Run()
}

func splitArgs(args []string) ([]string, []string) {
	var i int
	for j, a := range args {
		x := strings.IndexByte(a, "=")
		if x < 0 {
			i = j + 1
			break
		}
	}
	return args[:i], args[i:]
}

func printEnv() {
	for _, e := range os.Environ() {
		fmt.Fprintln(os.Stdout, e)
	}
}
