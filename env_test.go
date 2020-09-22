package tish

import (
	"reflect"
	"testing"
)

type EnvCase struct {
	ident string
	word  string
}

func TestEnv(t *testing.T) {
	ecs := []EnvCase{
		{ident: "HOME", word: "/home/tish"},
		{ident: "PATH", word: "/bin"},
		{ident: "FOO", word: "foo"},
		{ident: "BAR", word: "bar"},
	}
	e1 := EmptyEnv()
	for i := 0; i < 2; i++ {
		e1.Define(ecs[i].ident, ecs[i].word)
	}
	e2 := EnclosedEnv(e1)
	for i := 2; i < len(ecs); i++ {
		e2.Define(ecs[i].ident, ecs[i].word)
	}
	if n := len(e1.Environ()); n != 2 {
		t.Errorf("environ: want 2 items, got %d (%s)", n, e1.Environ())
	}
	if n := len(e2.Environ()); n != 4 {
		t.Errorf("environ: want 4 items, got %d (%s)", n, e2.Environ())

	}
	testResolve(t, e2, ecs)
	testResolve(t, e2, ecs)
}

func testResolve(t *testing.T, env Environment, ecs []EnvCase) {
	t.Helper()
	for _, e := range ecs {
		w := env.Resolve(e.ident)
		if w == "" {
			t.Errorf("%s: want %s, got empty word", e.ident, e.word)
			continue
		}
		if !reflect.DeepEqual(w, e.word) {
			t.Errorf("%s: word mismatched! want %s, got %s", e.ident, e.word, w)
		}
	}
}

func testDelete(t *testing.T, env *Env, ecs []EnvCase) {
	t.Helper()
	for _, e := range ecs {
		env.Delete(e.ident)
	}
	for _, e := range ecs {
		w := env.Resolve(e.ident)
		if w != "" {
			t.Errorf("%s: want empty word, got %s", e.ident, e.word)
		}
	}
}

func makeWord(str string) Word {
	tok := Token{
		Literal: str,
		Type:    TokLiteral,
	}
	return Literal{
		token: tok,
	}
}
