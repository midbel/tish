package tish

import (
  "testing"
)

func TestEnv(t *testing.T) {
  p := NewEnvironment()
  p.Define("HOME", []string{"/home/midbel"})
  p.Define("PATH", []string{"/bin:/usr/bin"})

  e := NewEnclosedEnvironment(p)
  e.Define("FOO", []string{"FOO"})
  e.Define("BAR", []string{"BAR"})

  env := e.Environ()
  if len(env) != 4 {
    t.Fatalf("expected global environ to have 4 items! got %d", len(env))
  }
  env = e.LocalEnviron()
  if len(env) != 2 {
    t.Fatalf("expected local environ to have 2 items! got %d", len(env))
  }

  e.SetReadOnly("BAR", true)
  if err := e.Define("BAR", []string{"ERROR"}); err == nil {
    t.Fatalf("define BAR should fail - readonly variable")
  }
  e.SetReadOnly("BAR", false)
  if err := e.Define("BAR", []string{"ERROR"}); err != nil {
    t.Fatalf("define BAR should succeed - read/write variable - %s", err)
  }

  e.Clear()
  env = e.Environ()
  if len(env) != 2 {
    t.Fatalf("expected global environ to have 2 items! got %d", len(env))
  }
  env = e.LocalEnviron()
  if len(env) != 0 {
    t.Fatalf("expected local environ to be empty! got %d", len(env))
  }
}
