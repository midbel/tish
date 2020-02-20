package tish

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
	"os"
	"path/filepath"
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
		Input  string
		Want   string
		File   string
		Stream *bytes.Buffer
	}{
		{
			Input:  `echo`,
			Want:   "",
			Stream: &sout,
		},
		{
			Input:  `echo foobar`,
			Want:   "foobar",
			Stream: &sout,
		},
		{
			Input:  `echo $HOME`,
			Want:   "/home/midbel",
			Stream: &sout,
		},
		{
			Input:  `echo '$HOME'`,
			Want:   "$HOME",
			Stream: &sout,
		},
		{
			Input:  `echo pre-" <$HOME> "-post`,
			Want:   "pre- </home/midbel> -post",
			Stream: &sout,
		},
		{
			Input:  `echo pre-{foo,bar}-post`,
			Want:   "pre-foo-post pre-bar-post",
			Stream: &sout,
		},
		{
			Input:  `echo foobar $(( 1 + (2*3)))`,
			Want:   "foobar 7",
			Stream: &sout,
		},
		{
			Input:  `echo foo; echo bar`,
			Want:   "foo\nbar",
			Stream: &sout,
		},
		{
			Input:  `echo foo && echo bar `,
			Want:   "foo\nbar",
			Stream: &sout,
		},
		{
			Input:  `echo foo || echo bar`,
			Want:   "foo",
			Stream: &sout,
		},
		{
			Input:  `printf "%s-%s" foo bar`,
			Want:   "foo-bar",
			Stream: &sout,
		},
		{
			Input:  `echo foo bar | echo -i`,
			Want:   "foo bar",
			Stream: &sout,
		},
		{
			Input:  `FOO=foobar; echo $FOO`,
			Want:   "foobar",
			Stream: &sout,
		},
		{
			Input:  `echo -i < testdata/foo.txt`,
			Want:   "foo",
			Stream: &sout,
		},
		{
			Input: `echo bar > testdata/bar.txt~`,
			Want:  "bar",
			File:  "testdata/bar.txt~",
		},
		{
			Input: `echo bar >> testdata/bar.txt~`,
			Want:  "bar\nbar",
			File:  "testdata/bar.txt~",
		},
		{
			Input: `echo foobar &> testdata/both.txt~`,
			Want:  "foobar",
			File:  "testdata/both.txt~",
		},
		{
			Input: `help -h 2> testdata/help.txt~`,
			Want:  "print help text for a builtin command\nusage: help builtin",
			File:  "testdata/help.txt~",
		},
		{
			Input:  `help -h`,
			Want:   "print help text for a builtin command\nusage: help builtin",
			Stream: &serr,
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
			got = d.Stream.String()
		} else {
			str, err := ioutil.ReadFile(d.File)
			if err != nil {
				t.Errorf("%s: fail read %s: %s", d.Input, d.File, err)
				continue
			}
			got = string(str)
		}
		got = strings.TrimSpace(got)
		if got != d.Want {
			t.Errorf("%s: values mismatched! want %s, got %s", d.Input, d.Want, got)
		}
	}
}
