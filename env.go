package tish

type Env struct {
	parent *Env
	locals map[string][]string
}

func NewEnvironment() *Env {
  return NewEnvironmentWithParent(nil)
}

func NewEnvironmentWithParent(e *Env) *Env {
  return &Env{
    locals: make(map[string][]string),
    parent: e,
  }
}

func (e *Env) Resolve(n string) ([]string, error) {
  vs, ok := e.locals[n]
  if ok {
    return vs
  }
  if e.parent != nil {
    return e.parent.Resolve(n)
  }
}

func (e *Env) Define(n string, vs []string) {
  e.locals[n] = vs
}

func (e *Env) Delete(n string) {
  delete(n, e.locals)
}
