package tish

import (
  "reflect"
  "testing"
)

type EnvCase struct {
  ident string
  word  Word
}

func TestEnv(t *testing.T) {
  ecs := []EnvCase{
    {ident: "HOME", word: makeWord("/home/tish")},
    {ident: "PATH", word: makeWord("/bin")},
    {ident: "FOO", word: makeWord("foo")},
    {ident: "BAR", word: makeWord("bar")},
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
    t.Errorf("environ: want %d, got %d", 2, n)
  }
  if n := len(e2.Environ()); n != 4 {
    t.Errorf("environ: want %d, got %d", 4, n)
  }
  testResolve(t, e2, ecs)
  testResolve(t, e2.Copy(), ecs)
  testUnwrap(t, e2, ecs[:2])
  testDelete(t, e2.Copy(), ecs)
  testResolve(t, e2, ecs)
}

func testResolve(t *testing.T, env *Env, ecs []EnvCase) {
  t.Helper()
  for _, e := range ecs {
    w := env.Resolve(e.ident)
    if w.IsZero() {
      t.Errorf("%s: want %s, got empty word", e.ident, e.word)
      continue
    }
    if !reflect.DeepEqual(w, e.word) {
      t.Errorf("%s: word mismatched! want %s, got %s", e.ident, e.word, w)
    }
  }
}

func testUnwrap(t *testing.T, env *Env, ecs []EnvCase) {
  t.Helper()
  env = env.Unwrap()
  if env.parent != nil {
    t.Errorf("unwrap: parent should be nil")
    return
  }
  testResolve(t, env.Unwrap(), ecs)
}

func testDelete(t *testing.T, env *Env, ecs []EnvCase) {
  t.Helper()
  for _, e := range ecs {
    env.Delete(e.ident)
  }
  for _, e := range ecs {
    w := env.Resolve(e.ident)
    if !w.IsZero() {
      t.Errorf("%s: want empty word, got %s", e.ident, e.word)
    }
  }
}

func makeWord(str string) Word {
  tok := Token{
    Literal: str,
    Type: TokLiteral,
  }
  return Word{
    tokens: []Token{tok},
  }
}
