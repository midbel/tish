package tish

import (
	"strings"
)

func ExampleExecuteWithEnv() {
	env := NewEnvironment()
	env.Set("HOME", []string{"/home/midbel"})

	scripts := []string{
		`echo`,
		`echo foobar`,
		`echo $HOME`,
		`echo '$HOME'`,
		`echo pre-" <$HOME> "-post`,
		`echo pre-{foo,bar}-post`,
		`echo foo $(( 1 + (2*3)))`,
		`printf "%s-%s" foo bar`,
		`echo foo; echo bar`,
		`echo foo && echo bar`,
		`echo foo || echo bar`,
	}
	for _, s := range scripts {
		if err := ExecuteWithEnv(strings.NewReader(s), env); err != nil {
			break
		}
	}
	// Output:
	//
	// foobar
	// /home/midbel
	// $HOME
	// pre- </home/midbel> -post
	// pre-foo-post pre-bar-post
	// foo 7
	// foo-bar
	// foo
	// bar
	// foo
	// bar
	// foo
}
