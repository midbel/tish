package tish

import (
	"flag"
	"io"
	"os"
	"strings"
)

func Run() error {
	sh := DefaultShell()

	flag.BoolVar(&sh.Verbose, "v", false, "print commands that will be executed on stderr")
	flag.BoolVar(&sh.Dry, "n", false, "dry run")
	var (
		profile = flag.String("r", DefaultProfile, "initialize shell from given scripts")
		cmdline = flag.Bool("c", false, "read command from the command string")
	)
	flag.Parse()

	if r, err := os.Open(*profile); err == nil {
		err := sh.Execute(r)
		r.Close()
		if err != nil {
			return err
		}
	}

	var r io.Reader
	if *cmdline {
		r = strings.NewReader(flag.Arg(0))
	} else {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}

	for i := 1; i < flag.NArg(); i++ {
		sh.Args = append(sh.Args, flag.Arg(i))
	}

	return sh.Execute(r)
}
