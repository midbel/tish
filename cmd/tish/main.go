package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/midbel/tish"
	"github.com/midbel/tish/internal/parser"
	"github.com/midbel/tish/internal/token"
)

type lockWriter struct {
	mu sync.Mutex
	io.Writer
}

func Lock(w io.Writer) io.Writer {
	return &lockWriter{
		Writer: w,
	}
}

func (w *lockWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Writer.Write(b)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Kill, os.Interrupt)
		<-sig
		cancel()
		close(sig)
	}()
	var (
		cwd      = flag.String("c", ".", "set working directory")
		name     = flag.String("n", "tish", "script name")
		echo     = flag.Bool("e", false, "echo each command before executing")
		scan     = flag.Bool("s", false, "scan script")
		parse    = flag.Bool("p", false, "parse script")
		inline   = flag.Bool("i", false, "read script from arguments")
		builddir = flag.String("b", "", "directory where additional builtin can be found")
	)
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "no enough argument supplied")
		os.Exit(1)
	}

	var err error
	switch {
	case *scan:
		err = scanScript(flag.Arg(0), *inline)
	case *parse:
		err = parseScript(flag.Arg(0), *inline)
	default:
	}
	if *scan || *parse {
		var code int
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			code = 2
		}
		os.Exit(code)
		return
	}

	options := []tish.ShellOption{
		tish.WithCwd(*cwd),
		tish.WithStdin(os.Stdin),
		tish.WithStdout(Lock(os.Stdout)),
		tish.WithStderr(Lock(os.Stderr)),
		tish.WithBuiltin(*builddir),
	}
	if *echo {
		options = append(options, tish.WithEcho())
	}

	sh, err := tish.New(options...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var args []string
	if flag.NArg() > 1 {
		args = flag.Args()
		args = args
	}
	if *inline {
		err = sh.Execute(ctx, flag.Arg(0), *name, args)
	} else {
		r, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		defer r.Close()
		err = sh.Run(ctx, r, filepath.Base(flag.Arg(0)), args)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to execute command: %s => %s", flag.Arg(0), err)
		fmt.Fprintln(os.Stderr)
	}
	sh.Exit()
}

func parseScript(script string, inline bool) error {
	var r io.Reader
	if inline {
		r = strings.NewReader(script)
	} else {
		f, err := os.Open(script)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	p := parser.NewParser(r)
	for {
		ex, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		_ = ex
	}
	return nil
}

func scanScript(script string, inline bool) error {
	var r io.Reader
	if inline {
		r = strings.NewReader(script)
	} else {
		f, err := os.Open(script)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	scan := parser.Scan(r)
	for i := 1; ; i++ {
		tok := scan.Scan()
		fmt.Fprintf(os.Stdout, "%3d: %s", i, tok)
		fmt.Fprintln(os.Stdout)
		if tok.Type == token.EOF || tok.Type == token.Invalid {
			break
		}
	}
	return nil
}
