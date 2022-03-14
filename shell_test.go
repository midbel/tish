package tish_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/midbel/tish"
)

const cmd = "go mod graph"

type stdio struct {
	Out bytes.Buffer
	Err bytes.Buffer
}

func (s *stdio) Reset() {
	s.Out.Reset()
	s.Err.Reset()
}

type ShellCase struct {
	Script string
	Out    []string
	Err    []string
	Args   []string
}

func TestShellBis(t *testing.T) {
	data := []ShellCase{
		{
			Script: `echo foobar`,
			Out:    []string{"foobar"},
		},
		{
			Script: `FOO=foobar; echo ${FOO} | cut -f 1 -d 'b'`,
			Out:    []string{"foo"},
		},
	}
	for _, d := range data {
		t.Run(d.Script, func(t *testing.T) {
			pair := Pair(d.Out, d.Err)
			sh, err := createShell(pair.Out, pair.Err)
			if err != nil {
				t.Fatalf("fail to create shell: %s", err)
			}
			if err := sh.Execute(context.TODO(), d.Script, "test", d.Args); err != nil {
				t.Fatalf("error while executing script: %s", err)
			}
		})
	}
}

func TestShell(t *testing.T) {
	var (
		sio     stdio
		sh, err = createShell(&sio.Out, &sio.Err)
	)
	if err != nil {
		t.Fatalf("unexpected error creating shell: %s", err)
	}
	t.Run("default", func(t *testing.T) {
		defer sio.Reset()

		executeScript(t, sh, cmd, &sio)
	})
	t.Run("redirection", func(t *testing.T) {
		defer sio.Reset()

		executeScript(t, sh, "echo foobar > testdata/foobar.txt; if [[ -s testdata/foobar.txt ]]; then echo ok fi", &sio)
		os.Remove("testdata/foobar.txt")
	})
	t.Run("alias", func(t *testing.T) {
		defer sio.Reset()

		sh.Alias("showgraph", cmd)
		executeScript(t, sh, "showgraph", &sio)
	})
	t.Run("assign", func(t *testing.T) {
		defer sio.Reset()

		executeScript(t, sh, "foobar = foobar; echo ${foobar} | cut -f 1 -d 'b'", &sio)
	})
	t.Run("conditional", func(t *testing.T) {
		defer sio.Reset()

		executeScript(t, sh, "true && echo foo", &sio)
		executeScript(t, sh, "false && echo foo; echo foo", &sio)
		executeScript(t, sh, "false || echo bar", &sio)
		executeScript(t, sh, "echo foo || echo bar", &sio)
	})
	t.Run("for-loop", func(t *testing.T) {
		defer sio.Reset()

		sh.Define("empty", []string{})
		executeScript(t, sh, "for label in foo bar; do echo $label; done", &sio)
		executeScript(t, sh, "for label in foo bar; do echo $label; break; done", &sio)
		executeScript(t, sh, "for label in foo bar; do echo $label; continue; done", &sio)
		executeScript(t, sh, "for label in $empty; do echo $label; else echo empty; done", &sio)
	})
	t.Run("test", func(t *testing.T) {
		defer sio.Reset()

		executeScript(t, sh, "if [[ -d testdata ]]; then echo ok else echo ko fi", &sio)
		executeScript(t, sh, "if [[ ! -d testdata ]]; then echo ko else echo ok fi", &sio)
	})
}

func executeScript(t *testing.T, sh *tish.Shell, script string, sio *stdio) {
	t.Helper()

	err := sh.Execute(context.TODO(), script, "test", nil)
	if err != nil {
		t.Fatalf("unexpected error executing command: %s", err)
	}
	t.Logf("length stdout: %d", sio.Out.Len())
	t.Logf("length stderr: %d", sio.Err.Len())
	if sio.Out.Len() == 0 {
		t.Errorf("stdout is empty")
	}
	if sio.Err.Len() != 0 {
		t.Errorf("stderr is not empty")
	}
}

func createShell(out, err io.Writer) (*tish.Shell, error) {
	options := []tish.ShellOption{
		tish.WithStdout(out),
		tish.WithStderr(err),
	}
	return tish.New(options...)
}

type stdpair struct {
	Out io.Writer
	Err io.Writer
}

func Pair(out, err []string) stdpair {
	return stdpair{
		Out: empty(out),
		Err: empty(err),
	}
}

type writer struct {
	curr int
	want []string
}

func empty(values []string) io.Writer {
	return &writer{
		want: values,
	}
}

func (w *writer) Write(b []byte) (int, error) {
	if w.curr >= len(w.want) {
		return 0, fmt.Errorf("too many values written")
	}
	got := bytes.TrimSpace(b)
	if w.want[w.curr] != string(got) {
		return 0, fmt.Errorf("strings mismatched! want %s, got %s", w.want[w.curr], got)
	}
	w.curr++
	return len(b), nil
}
