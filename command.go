package tish

import (
	"fmt"
	"strings"
)

type Command interface {
	Execute() (int, error)
	Equal(Command) bool
	fmt.Stringer
}

type Word struct {
	tokens []Token
}

func (w Word) String() string {
	ws := make([]string, len(w.tokens))
	for i := range w.tokens {
		ws[i] = w.tokens[i].Literal
	}
	return strings.Join(ws, "")
}

func (w Word) Equal(other Word) bool {
	if len(w.tokens) != len(other.tokens) {
		return false
	}
	for i, t := range w.tokens {
		if !t.Equal(other.tokens[i]) {
			return false
		}
	}
	return true
}

type Simple struct {
	words []Word
}

func (s Simple) Execute() (int, error) {
	return 0, nil
}

func (s Simple) Equal(other Command) bool {
	i, ok := other.(Simple)
	if !ok {
		return false
	}
	if len(s.words) != len(i.words) {
		return false
	}
	for j, w := range s.words {
		if !w.Equal(i.words[j]) {
			return false
		}
	}
	return true
}

func (s Simple) String() string {
	ws := make([]string, len(s.words))
	for i := range s.words {
		ws[i] = s.words[i].String()
	}
	return fmt.Sprintf("simple(%s)", strings.Join(ws, " "))
}

type List struct {
	cmds []Command
}

func (i List) Execute() (int, error) {
	var (
		code int
		err  error
	)
	for _, c := range i.cmds {
		code, err = c.Execute()
	}
	return code, err
}

func (i List) Equal(other Command) bool {
	s, ok := other.(List)
	if !ok {
		return ok
	}
	for j, c := range i.cmds {
		if !c.Equal(s.cmds[j]) {
			return false
		}
	}
	return true
}

func (i List) String() string {
	ws := make([]string, len(i.cmds))
	for j, c := range i.cmds {
		ws[j] = c.String()
	}
	return fmt.Sprintf("list(%s)", strings.Join(ws, ", "))
}

type And struct {
	left  Command
	right Command
}

func (a And) Execute() (int, error) {
	e, err := a.left.Execute()
	if e == 0 && err == nil {
		e, err = a.right.Execute()
	}
	return e, err
}

func (a And) Equal(other Command) bool {
	i, ok := other.(And)
	if !ok {
		return ok
	}
	return a.left.Equal(i.left) && a.right.Equal(i.right)
}

func (a And) String() string {
	return fmt.Sprintf("and(%s, %s)", a.left, a.right)
}

type Or struct {
	left  Command
	right Command
}

func (o Or) Execute() (int, error) {
	e, err := o.left.Execute()
	if e == 0 && err == nil {
		return e, err
	}
	return o.right.Execute()
}

func (o Or) Equal(other Command) bool {
	i, ok := other.(Or)
	if !ok {
		return ok
	}
	return o.left.Equal(i.left) && o.right.Equal(i.right)
}

func (o Or) String() string {
	return fmt.Sprintf("or(%s, %s)", o.left, o.right)
}

type If struct {
	cmd Command
	csq Command
	alt Command
}

func (i If) Execute() (int, error) {
	e, err := i.cmd.Execute()
	if e == 0 && err == nil {
		return i.csq.Execute()
	}
	if i.alt != nil {
		e, err = i.alt.Execute()
	}
	return e, err
}

func (i If) Equal(other Command) bool {
	j, ok := other.(If)
	if !ok {
		return ok
	}
	return i.cmd.Equal(j.cmd) && i.csq.Equal(j.csq) && i.alt.Equal(j.alt)
}

func (i If) String() string {
	return fmt.Sprintf("if(cmd: %s, csq: %s, alt: %s)", i.cmd, i.csq, i.alt)
}

type Until struct {
	cmd  Command
	body Command
}

func (u Until) Execute() (int, error) {
	var (
		code int
		err  error
	)
	for {
		if e, err := u.cmd.Execute(); e == 0 && err == nil {
			break
		}
		code, err = u.body.Execute()
	}
	return code, err
}

func (u Until) Equal(other Command) bool {
	i, ok := other.(Until)
	if !ok {
		return ok
	}
	return u.cmd.Equal(i.cmd) && u.body.Equal(i.body)
}

func (u Until) String() string {
	return fmt.Sprintf("until(cmd: %s, body: %s)", u.cmd, u.body)
}

type While struct {
	cmd  Command
	body Command
}

func (w While) Execute() (int, error) {
	var (
		code int
		err  error
	)
	for {
		if e, err := w.cmd.Execute(); e != 0 || err != nil {
			break
		}
		code, err = w.body.Execute()
	}
	return code, err
}

func (w While) Equal(other Command) bool {
	i, ok := other.(While)
	if !ok {
		return ok
	}
	return w.cmd.Equal(i.cmd) && w.body.Equal(i.body)
}

func (w While) String() string {
	return fmt.Sprintf("while(cmd: %s, body: %s)", w.cmd, w.body)
}
