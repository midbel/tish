package tish

import (
	"io"
)

type ShellOption func(*Shell) error

func WithFinder(find CommandFinder) ShellOption {
	return func(s *Shell) error {
		s.find = find
		return nil
	}
}

func WithStdin(r io.Reader) ShellOption {
	return func(s *Shell) error {
		s.SetIn(r)
		return nil
	}
}

func WithStdout(w io.Writer) ShellOption {
	return func(s *Shell) error {
		s.SetOut(w)
		return nil
	}
}

func WithStderr(w io.Writer) ShellOption {
	return func(s *Shell) error {
		s.SetErr(w)
		return nil
	}
}

func WithEcho() ShellOption {
	return func(s *Shell) error {
		s.echo = true
		return nil
	}
}

func WithVar(ident string, values ...string) ShellOption {
	return func(s *Shell) error {
		if s.locals == nil {
			s.locals = EmptyEnv()
		}
		return s.locals.Define(ident, values)
	}
}

func WithCwd(dir string) ShellOption {
	return func(s *Shell) error {
		return s.Chdir(dir)
	}
}

func WithEnv(e Environment) ShellOption {
	return func(s *Shell) error {
		s.locals = EnclosedEnv(e)
		return nil
	}
}

func WithExport(vs map[string]string) ShellOption {
	return func(s *Shell) error {
		for k, v := range vs {
			s.Export(k, v)
		}
		return nil
	}
}

func WithAlias(vs map[string]string) ShellOption {
	return func(s *Shell) error {
		for k, v := range vs {
			s.Alias(k, v)
		}
		return nil
	}
}
