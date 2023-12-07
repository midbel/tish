package tish_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/midbel/tish"
)

func TestParser(t *testing.T) {
	files := []string{
		"basic.sh",
		"builtins.sh",
		"quote.sh",
		"redirect.sh",
	}
	for _, f := range files {
		testParser(t, f)
	}
}

func testParser(t *testing.T, file string) {
	t.Helper()
	r, err := os.Open(filepath.Join("testdata", file))
	if err != nil {
		t.Errorf("fail to open file %s", file)
		return
	}
	p, err := tish.New(r)
	if err != nil {
		t.Errorf("parser can not be created: %s", err)
		return
	}
	for {
		_, err := p.Parse()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Errorf("error parsing command: %s", err)
		}
	}
}
