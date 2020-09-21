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
