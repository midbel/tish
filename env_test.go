package tish

import (
  "errors"
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

  copyEnv(t, e)
  setReadWrite(t, e)
  clearEnv(t, e)
}

func copyEnv(t *testing.T, e *Env) {
  t.Helper()

  e.Define("TEST", []string{"TEST"})
  defer e.Del("TEST")
  e.SetReadOnly("TEST", true)

  c := e.Copy()
  c.Define("STR", []string{"STR"})

  if _, err := e.Resolve("STR"); !errors.Is(err, ErrNotDefined) {
    t.Fatalf("STR variable should not be defined in env: %s", err)
  }
  if _, err := c.Resolve("STR"); err != nil {
    t.Fatalf("STR variable should be defined in copy: %s", err)
  }

  if err := c.Define("TEST", []string{"ERROR"}); err == nil {
    t.Fatalf("copy should preserve read only flag on variable")
  }
}

func setReadWrite(t *testing.T, e *Env) {
  t.Helper()

  e.SetReadOnly("BAR", true)
  if err := e.Define("BAR", []string{"ERROR"}); err == nil {
    t.Fatalf("define BAR should fail - readonly variable")
  }
  e.SetReadOnly("BAR", false)
  if err := e.Define("BAR", []string{"ERROR"}); err != nil {
    t.Fatalf("define BAR should succeed - read/write variable - %s", err)
  }
}

func clearEnv(t *testing.T, e *Env) {
  t.Helper()

  e.Clear()
  env := e.Environ()
  if len(env) != 2 {
    t.Fatalf("expected global environ to have 2 items! got %d", len(env))
  }
  env = e.LocalEnviron()
  if len(env) != 0 {
    t.Fatalf("expected local environ to be empty! got %d", len(env))
  }
}
