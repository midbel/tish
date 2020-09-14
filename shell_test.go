package tish

import (
	"bytes"
	"strings"
	"testing"
)

type OutputCase struct {
	Input string
	Want  string
}

func TestAssign(t *testing.T) {
	data := []OutputCase{
		{
			Input: "FOO=foo; echo $FOO",
			Want:  "foo\n",
		},
		{
			Input: "FOO=foo; BAR=bar; echo \"$FOO $BAR\"",
			Want:  "foo bar\n",
		},
		{
			Input: "FOO=foo; BAR=bar; echo '$FOO $BAR'",
			Want:  "$FOO $BAR\n",
		},
	}
	testOutputCase(t, data, true)
}

func TestLoop(t *testing.T) {
	t.Run("for", testFor)
	// t.Run("while", testWhile)
	// t.Run("until", testUntil)
}

func testFor(t *testing.T) {
	data := []OutputCase{
		{
			Input: "for VAR in foo bar; do echo $VAR; done",
			Want:  "foo\nbar\n",
		},
	}
	testOutputCase(t, data, false)
}

func TestConditionals(t *testing.T) {
	t.Run("if", testIf)
	// t.Run("case", testCase)
}

func testIf(t *testing.T) {
	data := []OutputCase{
		{
			Input: "if true; then echo foo; fi",
			Want:  "foo\n",
		},
		{
			Input: "if false; then echo foo; else echo bar; fi",
			Want:  "bar\n",
		},
	}
	testOutputCase(t, data, false)
}

func TestBuiltins(t *testing.T) {
	t.Run("echo", testEcho)
	t.Run("true", testTrue)
	t.Run("false", testFalse)
	t.Run("exit", testExit)
}

func testEcho(t *testing.T) {
	data := []OutputCase{
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
		{
			Input: "false || echo foo",
			Want:  "foo\n",
		},
	}
	testOutputCase(t, data, true)
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
		t.Errorf("true: unexpected exit code! want %d, got %d", ExitOk, exit)
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
	if exit != ExitKo {
		t.Errorf("false: unexpected exit code! want %d, got %d", ExitKo, exit)
	}
}

func testExit(t *testing.T) {
	data := []struct {
		Input string
		Exit  int
	}{
		{Input: "exit", Exit: 0},
		{Input: "false; exit", Exit: 1},
		{Input: "true; exit", Exit: 0},
		{Input: "exit 0", Exit: 0},
		{Input: "exit 1", Exit: 1},
		{Input: "exit 255", Exit: 255},
		{Input: "exit -- -1", Exit: 1},
		{Input: "exit && echo foo", Exit: 0},
		{Input: "exit 1; echo foo", Exit: 1},
		{Input: "exit || echo foo", Exit: 0},
		{Input: "exit 5 || echo foo", Exit: 5},
		{Input: "exit 5 || echo foo", Exit: 5},
	}

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
	for _, d := range data {
		stdout.Reset()
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
		if exit != d.Exit {
			t.Errorf("%s: exit code mismatched! want %d, got %d", d.Input, d.Exit, exit)
			continue
		}
		if stdout.Len() > 0 || stdout.String() != "" {
			t.Errorf("%s: expected stdout to be empty! got %s", d.Input, stdout.String())
		}
	}
}

func testOutputCase(t *testing.T, data []OutputCase, chexit bool) {
	t.Helper()

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
			t.Errorf("%s: unexpected error: %s", d.Input, err)
			continue
		}
		if chexit && exit != ExitOk {
			t.Errorf("%s: unexpected exit code! want %d, got %d", d.Input, ExitOk, exit)
			continue
		}
		if got := stdout.String(); got != d.Want {
			g, w := strings.TrimSpace(got), strings.TrimSpace(d.Want)
			t.Errorf("%s: want %s, got %s", d.Input, w, g)
		}
	}
}
