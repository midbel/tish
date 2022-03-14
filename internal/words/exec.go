package words

import (
	"errors"

	"github.com/midbel/tish/internal/token"
)

var (
	ErrBreak    = errors.New(token.KwBreak)
	ErrContinue = errors.New(token.KwContinue)
)

type Executer interface{}

type ExecSimple struct {
	Expander
	Redirect []ExpandRedirect
}

func CreateSimple(ex Expander) ExecSimple {
	return ExecSimple{
		Expander: ex,
	}
}

type ExecAssign struct {
	Ident string
	Expander
}

func CreateAssign(ident string, ex Expander) ExecAssign {
	return ExecAssign{
		Ident:    ident,
		Expander: ex,
	}
}

type ExecList []Executer

func (e ExecList) Executer() Executer {
	if len(e) == 1 {
		return e[0]
	}
	return e
}

type ExecAnd struct {
	Left  Executer
	Right Executer
}

func CreateAnd(left, right Executer) ExecAnd {
	return ExecAnd{
		Left:  left,
		Right: right,
	}
}

type ExecOr struct {
	Left  Executer
	Right Executer
}

func CreateOr(left, right Executer) ExecOr {
	return ExecOr{
		Left:  left,
		Right: right,
	}
}

type ExecPipe struct {
	List []PipeItem
}

func CreatePipe(list []PipeItem) ExecPipe {
	return ExecPipe{
		List: list,
	}
}

type PipeItem struct {
	Executer
	Both bool
}

func CreatePipeItem(ex Executer, both bool) PipeItem {
	return PipeItem{
		Executer: ex,
		Both:     both,
	}
}

type ExecSubshell []Executer

func (e ExecSubshell) Executer() Executer {
	if len(e) == 1 {
		return e[0]
	}
	return e
}

type ExecBreak struct{}

type ExecContinue struct{}

type ExecFor struct {
	Ident string
	List  []Expander
	Body  Executer
	Alt   Executer
}

func (e ExecFor) Expand(env Environment, _ bool) ([]string, error) {
	var list []string
	for i := range e.List {
		str, err := e.List[i].Expand(env, false)
		if err != nil {
			return nil, err
		}
		list = append(list, str...)
	}
	return list, nil
}

type ExecWhile struct {
	Cond Executer
	Body Executer
	Alt  Executer
}

type ExecUntil struct {
	Cond Executer
	Body Executer
	Alt  Executer
}

type ExecIf struct {
	Cond Executer
	Csq  Executer
	Alt  Executer
}

type ExecCase struct {
	Word    Expander
	List    []ExecClause
	Default Executer
}

type ExecClause struct {
	List []Expander
	Body Executer
}

type ExecTest struct {
	Tester
}
