package tish

import (
  "fmt"
  "io"
  "os"
  "os/exec"
  "sort"
)

var builtins = []string{
  "export",
  "date",
  "alias",
  "unalias",
  "pwd",
  "cd",
  "vars",
  "env",
  // "echo",
}

func init() {
  sort.Strings(builtins)
}

func Execute(r io.Reader) error {
  return ExecuteWithEnv(r, NewEnvironment())
}

func ExecuteWithEnv(r io.Reader, env *Env) error {
  ws, err := Parse(r)
  if err != nil {
    return err
  }
  env = NewEnclosedEnvironment(env)
  switch w := ws.(type) {
  case Literal:
    err = executeLiteral(w, env)
  case List:
    err = executeList(w, env)
  default:
    err = fmt.Errorf("exec: %T can not be executed", w)
  }
  return err
}

func executeList(i List, e *Env) error {
  var err error
  switch i.kind {
  case kindSimple:
    err = executeSimple(i, e)
  case kindPipe:
    err = executePipeline(i, e)
  case kindSeq:
    err = executeSequence(i.words, e)
  case kindAnd:
    err = executeAnd(i.words, e)
  case kindOr:
    err = executeOr(i.words, e)
  default:
    err = fmt.Errorf("exec: %s can not be executed", i.kind)
  }
  return err
}

func executeOr(ws []Word, e *Env) error {
  var err error
  for _, w := range ws {
    switch w := w.(type) {
    case Literal:
      err = executeLiteral(w, e)
    case List:
      err = executeList(w, e)
    default:
      return fmt.Errorf("exec: %T can not be executed", w)
    }
    if err == nil {
      break
    }
  }
  return err
}

func executeAnd(ws []Word, e *Env) error {
  var err error
  for _, w := range ws {
    switch w := w.(type) {
    case Literal:
      err = executeLiteral(w, e)
    case List:
      err = executeList(w, e)
    default:
      return fmt.Errorf("exec: %T can not be executed", w)
    }
    if err != nil {
      break
    }
  }
  return err
}

func executePipeline(i List, e *Env) error {
  return nil
}

func executeSequence(ws []Word, e *Env) error {
  var err error
  for _, w := range ws {
    switch w := w.(type) {
    case Literal:
      err = executeLiteral(w, e)
    case List:
      err = executeList(w, e)
    default:
      return fmt.Errorf("exec: %T can not be executed", w)
    }
  }
  return err
}

func executeSimple(w Word, e *Env) error {
  vs, err := w.Expand(e)
  if err != nil {
    return err
  }
  fmt.Printf("%q\n", vs[1:])
  cmd := exec.Command(vs[0], vs[1:]...)
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr
  return cmd.Run()
}

func executeLiteral(i Literal, e *Env) error {
  vs, err := i.Expand(e)
  if err != nil || len(vs) == 0 {
    return err
  }
  cmd := exec.Command(vs[0], vs[1:]...)
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr
  return cmd.Run()
}
