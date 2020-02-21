package tish

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteWithEnv(t *testing.T) {
	defer func() {
		filepath.Walk("testdata", func(f string, i os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if i.Mode().IsRegular() && filepath.Ext(f) == ".txt~" {
				os.Remove(f)
			}
			return nil
		})
	}()

	var (
		env  = NewEnvironment()
		sout bytes.Buffer
		serr bytes.Buffer
	)
	env.Set("HOME", []string{"/home/midbel"})

	stdin = bytes.NewReader(make([]byte, 64))
	stdout = &sout
	stderr = &serr

	data := []struct {
		Input string
		Out   string
		Err   string
		File  string
	}{
		{
			Input: `echo`,
			Out:   "",
			Err:   "",
		},
		{
			Input: `echo foobar`,
			Out:   "foobar",
			Err:   "",
		},
		{
			Input: `echo $HOME`,
			Out:   "/home/midbel",
			Err:   "",
		},
		{
			Input: `echo '$HOME'`,
			Out:   "$HOME",
			Err:   "",
		},
		{
			Input: `echo pre-" <$HOME> "-post`,
			Out:   "pre- </home/midbel> -post",
			Err:   "",
		},
		{
			Input: `echo pre-{foo,bar}-post`,
			Out:   "pre-foo-post pre-bar-post",
			Err:   "",
		},
		{
			Input: `echo foobar $(( 1 + (2*3)))`,
			Out:   "foobar 7",
			Err:   "",
		},
		{
			Input: `echo foo; echo bar`,
			Out:   "foo\nbar",
			Err:   "",
		},
		{
			Input: `echo foo && echo bar `,
			Out:   "foo\nbar",
			Err:   "",
		},
		{
			Input: `echo foo || echo bar`,
			Out:   "foo",
			Err:   "",
		},
		{
			Input: `printf "%s-%s" foo bar`,
			Out:   "foo-bar",
			Err:   "",
		},
		{
			Input: `echo foo bar | echo -i`,
			Out:   "foo bar",
			Err:   "",
		},
		{
			Input: `FOO=foobar; echo $FOO`,
			Out:   "foobar",
			Err:   "",
		},
		{
			Input: `echo -i < testdata/foo.txt`,
			Out:   "foo",
			Err:   "",
		},
		{
			Input: `echo bar > testdata/bar.txt~`,
			Out:   "bar",
			Err:   "",
			File:  "testdata/bar.txt~",
		},
		{
			Input: `echo bar >> testdata/bar.txt~`,
			Out:   "bar\nbar",
			Err:   "",
			File:  "testdata/bar.txt~",
		},
		{
			Input: `echo foobar &> testdata/both.txt~`,
			Out:   "foobar",
			Err:   "",
			File:  "testdata/both.txt~",
		},
		{
			Input: `help -h 2> testdata/help.txt~`,
			Out:   "print help text for a builtin command\nusage: help builtin",
			Err:   "",
			File:  "testdata/help.txt~",
		},
		{
			Input: `help -h`,
			Out:   "",
			Err:   "print help text for a builtin command\nusage: help builtin",
		},
		{
			Input: `help -h |& echo -i`,
			Out:   "",
			Err:   "print help text for a builtin command\nusage: help builtin",
		},
	}
	for _, d := range data {
		sout.Reset()
		serr.Reset()

		err := ExecuteWithEnv(strings.NewReader(d.Input), env)
		if err != nil {
			t.Fatalf("%s: unexpected error: %s", d.Input, err)
			continue
		}

		var got string
		if d.File == "" {
			err := strings.TrimSpace(serr.String())
			if err != d.Err {
				t.Errorf("%s: stderr mismatched! want %s, got %s", d.Input, d.Err, err)
			}
			out := strings.TrimSpace(sout.String())
			if out != d.Out {
				t.Errorf("%s: stdout mismatched! want %s, got %s", d.Input, d.Out, out)
			}
		} else {
			str, err := ioutil.ReadFile(d.File)
			if err != nil {
				t.Errorf("%s: fail read %s: %s", d.Input, d.File, err)
				continue
			}
			got = strings.TrimSpace(string(str))
			if got != d.Out {
				t.Errorf("%s: values mismatched! want %s, got %s", d.Input, d.Out, got)
			}
		}
	}
}
