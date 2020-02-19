package tish

import (
	"strings"
	"testing"
	"bytes"
)

func TestExecuteWithEnv(t *testing.T) {
	var (
		env = NewEnvironment()
		sout bytes.Buffer
		serr bytes.Buffer
	)
	env.Set("HOME", []string{"/home/midbel"})

	stdin = bytes.NewReader(make([]byte, 64))
	stdout = &sout
	stderr = &serr

	data := []struct{
		Input string
		Want  string
	} {
		{
			Input: `echo`,
			Want: "",
		},
		{
			Input: `echo foobar`,
			Want: "foobar",
		},
		{
			Input: `echo $HOME`,
			Want: "/home/midbel",
		},
		{
			Input: `echo '$HOME'`,
			Want: "$HOME",
		},
		{
			Input: `echo pre-" <$HOME> "-post`,
			Want: "pre- </home/midbel> -post",
		},
		{
			Input: `echo pre-{foo,bar}-post`,
			Want: "pre-foo-post pre-bar-post",
		},
		{
			Input: `echo foobar $(( 1 + (2*3)))`,
			Want: "foobar 7",
		},
		{
			Input: `echo foo; echo bar`,
			Want: "foo\nbar",
		},
		{
			Input: `echo foo && echo bar `,
			Want: "foo\nbar",
		},
		{
			Input: `echo foo || echo bar`,
			Want: "foo",
		},
		{
			Input: `printf "%s-%s" foo bar`,
			Want: "foo-bar",
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

		if str := serr.String(); len(str) > 0 {
			t.Errorf("%s: expected stderr empty, got %s (%d)", d.Input, str, len(str))
			continue
		}
		got := strings.TrimSpace(sout.String())
		if got != d.Want {
			t.Errorf("%s: values mismatched! want %s, got %s", d.Input, d.Want, got)
		}
	}
}
