package tish

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuiltins(t *testing.T) {
	t.Run("echo", testEcho)
	t.Run("true", testTrue)
	t.Run("false", testFalse)
}

func testEcho(t *testing.T) {
	var (
		stdin  = bytes.NewReader(nil)
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	options := []Option{
		WithStdin(stdin),
		WithStdout(&stdout),
		WithStderr(&stderr),
	}
	data := []struct {
		Input string
		Want  string
	}{
		{
			Input: "echo foo bar",
			Want:  "foo bar\n",
		},
		{
			Input: "echo",
			Want:  "\n",
		},
		{
			Input: "true && echo bar",
			Want:  "bar\n",
		},
		// {
		// 	Input: "false || echo foo",
		// 	Want:  "foo\n",
		// },
	}
	for _, d := range data {
		stdout.Reset()
		stderr.Reset()
		s, err := NewShell(strings.NewReader(d.Input), options...)
		if err != nil {
			t.Errorf("%s: %s", d.Input, err)
			continue
		}
		exit, err := s.Execute()
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		if exit != ExitOk {
			t.Errorf("unexpected exit code! want 0, got %d", exit)
			continue
		}
		if got := stdout.String(); got != d.Want {
			t.Errorf("%[1]s: want %[2]s %[2]x, got %[3]s %[3]x", d.Input, d.Want, got)
		}
	}
}

func testTrue(t *testing.T) {
	var (
		stdin  = bytes.NewReader(nil)
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	options := []Option{
		WithStdin(stdin),
		WithStdout(&stdout),
		WithStderr(&stderr),
	}
	s, err := NewShell(strings.NewReader("true"), options...)
	if err != nil {
		t.Errorf("true: %s", err)
		return
	}
	exit, err := s.Execute()
	if err != nil {
		t.Errorf("true: unexpected error: %s", err)
		return
	}
	if exit != ExitOk {
		t.Errorf("true: unexpected exit code! want 0, got %d", exit)
	}
}

func testFalse(t *testing.T) {
	var (
		stdin  = bytes.NewReader(nil)
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	options := []Option{
		WithStdin(stdin),
		WithStdout(&stdout),
		WithStderr(&stderr),
	}
	s, err := NewShell(strings.NewReader("false"), options...)
	if err != nil {
		t.Errorf("false: %s", err)
		return
	}
	exit, err := s.Execute()
	if err != nil {
		t.Errorf("false: unexpected error: %s", err)
		return
	}
	if exit == ExitOk {
		t.Errorf("false: unexpected exit code! want 1, got %d", exit)
	}
}
