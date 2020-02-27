package tish

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type ShellCase struct {
	Input string
	Out   string
	Err   string
	File  string
}

func TestShellExecute(t *testing.T) {
	t.Cleanup(func() {
		filepath.Walk("testdata", func(p string, i os.FileInfo, err error) error {
			if i.Mode().IsRegular() && filepath.Ext(p) == ".txt~" {
				os.Remove(p)
			}
			return err
		})
	})
	data := []ShellCase{
		{
			Input: `echo foo bar`,
			Out:   "foo bar",
		},
		{
			Input: `echo -h`,
			Err:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
		},
		{
			Input: `echo -i < testdata/foo.txt`,
			Out:   "foo",
		},
		{
			Input: `alias readfile="echo -i"; readfile < testdata/foo.txt`,
			Out:   "foo",
		},
		{
			Input: `echo`,
			Out:   "",
		},
		{
			Input: `FOO=FOO; BAR=BAR; echo $FOO $BAR`,
			Out:   "FOO BAR",
		},
		{
			Input: `FOO=FOO BAR=BAR env FOO BAR`,
			Out:   "FOO=FOO\nBAR=BAR",
		},
		{
			Input: `echo bar > testdata/bar.txt~`,
			Out:   "bar",
			File:  "testdata/bar.txt~",
		},
		{
			Input: `echo bar >> testdata/bar.txt~`,
			Out:   "bar\nbar",
			File:  "testdata/bar.txt~",
		},
		{
			Input: `echo -h 2> testdata/help.txt~`,
			Out:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
			File:  "testdata/help.txt~",
		},
		{
			Input: `echo foo bar | echo -i`,
			Out:   "foo bar",
		},
		{
			Input: `echo -h |& echo -i`,
			Out:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
		},
		{
			Input: `echo foobar >&2`,
			Err:   "foobar",
		},
		{
			Input: `echo -h 2>&1`,
			Out:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
		},
	}
	testShellCase(t, data)
}

func testShellCase(t *testing.T, data []ShellCase) {
	t.Helper()

	var (
		sin  = bytes.NewReader(nil)
		sout bytes.Buffer
		serr bytes.Buffer
	)

	trim := func(str string) string {
		return strings.TrimSpace(str)
	}

	for _, d := range data {
		sout.Reset()
		serr.Reset()

		sh := NewShell(sin, &sout, &serr)
		if err := sh.ExecuteString(d.Input); err != nil {
			t.Errorf("%s: unexpected error %s", d.Input, err)
			continue
		}
		if d.File != "" {
			str, err := ioutil.ReadFile(d.File)
			if err != nil {
				t.Errorf("%s: %s", d.Input, err)
				continue
			}
			if got := trim(string(str)); got != d.Out {
				t.Errorf("%s: file mismatched! want %s, got %s", d.Input, d.Out, got)
			}
			continue
		}
		if got := trim(sout.String()); got != d.Out {
			t.Errorf("%s: stdout mismatched! want %s, got %s", d.Input, d.Out, got)
		}
		if got := trim(serr.String()); got != d.Err {
			t.Errorf("%s: stderr mismatched! want %s, got %s", d.Input, d.Err, got)
		}
	}
}
