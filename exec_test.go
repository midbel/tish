package tish

import (
	"strings"
)

func ExampleExecute() {
	env := NewEnvironment()
	env.Set("HOME", []string{"/home/midbel"})

	ExecuteWithEnv(strings.NewReader("echo foobar"), env)
	ExecuteWithEnv(strings.NewReader("echo $HOME"), env)
	// Output:
	// foobar
	// /home/midbel
}
