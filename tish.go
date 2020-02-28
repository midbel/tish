package tish

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var DefaultProfile string

const (
	Version = "0.0.1"
	Tish    = "tish"
)

func init() {
	home, _ := os.UserHomeDir()
	DefaultProfile = filepath.Join(home, ".tishrc")
}

func Run() error {
	var (
		profile = flag.String("r", DefaultProfile, "initialize shell from given scripts")
		cmdline = flag.Bool("c", false, "read command from the command string")
	)
	flag.Parse()

	sh := DefaultShell()
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
