package tish_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/midbel/tish"
)

var list = []struct {
	Input   string
	Invalid bool
	Len     int
}{
	{
		Input: `echo foobar`,
		Len:   1,
	},
	{
		Input: `cat foo | grep -v bar > foobar.out 2> foobar.err`,
		Len:   1,
	},
	{
		Input: `echo "foobar"`,
		Len:   1,
	},
	{
		Input: `echo "${#foobar} - $(cat foo | grep bar)"`,
		Len:   1,
	},
	{
		Input: `echo pre-"foobar"-post`,
		Len:   1,
	},
	{
		Input: `echo pre-"$foobar"-post`,
		Len:   1,
	},
	{
		Input: `foobar="foo"; echo $foobar`,
		Len:   2,
	},
	{
		Input: `echo foobar | cat | cut -d b -f 1`,
		Len:   1,
	},
	{
		Input: `echo {{A,B,C,D},{001..5..1}}`,
		Len:   1,
	},
	{
		Input: `echo pre-{1..5}-post`,
		Len:   1,
	},
	{
		Input: `echo ${lower,,} && cat ${upper^^} || echo ${#foobar} `,
		Len:   1,
	},
	{
		Input: `echo ${foobar:2:5}; echo ${foobar#pre}; echo ${foobar%post}`,
		Len:   3,
	},
	{
		Input: `echo ${foobar//foo/bar}; echo ${foobar:<0:2}; echo ${foobar:> :2}`,
		Len:   3,
	},
	{
		Input: `echo $(cat $(echo ${file##.txt}))`,
		Len:   1,
	},
	{
		Input: `for ident in {1..5}; do echo $ident done`,
		Len:   1,
	},
	{
		Input: `for ident in $(seq 1 5 10); do echo $ident done`,
		Len:   1,
	},
	{
		Input: `for ident in {1..5}; do echo $ident else echo zero; done`,
		Len:   1,
	},
	{
		Input: `while true; do echo foo; done`,
		Len:   1,
	},
	{
		Input: `while true; do echo foo; break; done`,
		Len:   1,
	},
	{
		Input: `while true; do echo foo; continue; else echo "else"; done`,
		Len:   1,
	},
	{
		Input: `until true; do echo foo; else echo "else"; done`,
		Len:   1,
	},
	{
		Input: `if $foo; then echo foo; fi`,
		Len:   1,
	},
	{
		Input: `if $foo; then echo foo; else if $bar; then echo bar; else echo foobar; fi`,
		Len:   1,
	},
	{
		Input: `echo $((1+1))`,
		Len:   1,
	},
	{
		Input: `echo "$((1+1))"`,
		Len:   1,
	},
	{
		Input: `echo $((-1+(1*5)))`,
		Len:   1,
	},
	{
		Input: `echo $((VAR = 10; VAR+(1*5)))`,
		Len:   1,
	},
	{
		Input: `echo $((-foo+(1*5)))`,
		Len:   1,
	},
	{
		Input: `echo $((1+1; 2**2))`,
		Len:   1,
	},
	{
		Input: "echo foo\necho bar",
		Len:   2,
	},
	{
		Input: "[[ -z str && (file -eq other || file -ot $other)]]",
		Len:   1,
	},
	{
		Input: "[[ $var ]]",
		Len:   1,
	},
}

func TestParse(t *testing.T) {
	for _, in := range list {
		c := parse(t, in.Input, in.Invalid)
		if in.Len != c {
			t.Errorf("sequence mismatched! expected %d, got %d", in.Len, c)
		}
	}
}

func parse(t *testing.T, in string, invalid bool) int {
	t.Helper()
	var (
		p = tish.NewParser(strings.NewReader(in))
		c int
	)
	for {
		_, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if invalid && err == nil {
				t.Errorf("expected error parsing %q! got none", in)
				return -1
			}
			if !invalid && err != nil {
				t.Errorf("expected no error parsing %q! got %s", in, err)
				return -1
			}
		}
		c++
	}
	return c
}
