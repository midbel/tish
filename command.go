package tish

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrBreak    = errors.New(kwBreak)
	ErrContinue = errors.New(kwContinue)
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
	return fmt.Sprintf("word(%s)", strings.Join(ws, ""))
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
	env   []Assign
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
	if len(s.env) != len(i.env) {
		return false
	}
	for j, a := range s.env {
		if !a.Equal(i.env[j]) {
			return false
		}
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
	if len(i.cmds) != len(s.cmds) {
		return false
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

type Case struct {
	word    Word
	clauses []Command
}

func (c Case) Execute() (int, error) {
	return 0, nil
}

func (c Case) Equal(other Command) bool {
	i, ok := other.(Case)
	if !ok {
		return ok
	}
	if len(c.clauses) != len(i.clauses) {
		return false
	}
	for j, c := range c.clauses {
		if !c.Equal(i.clauses[j]) {
			return false
		}
	}
	return true
}

func (c Case) String() string {
	ws := make([]string, len(c.clauses))
	for i, c := range c.clauses {
		ws[i] = c.String()
	}
	return fmt.Sprintf("case(word: %s, body: %s)", c.word, ws)
}

type Clause struct {
	pattern  []Word
	body     Command
	op Token
}

func (c Clause) Execute() (int, error) {
	return c.body.Execute()
}

func (c Clause) Equal(other Command) bool {
	i, ok := other.(Clause)
	if !ok {
		return ok
	}
	if len(c.pattern) != len(i.pattern) {
		return false
	}
	for j, w := range c.pattern {
		if !w.Equal(i.pattern[j]) {
			return false
		}
	}
	if c.body != nil && i.body != nil {
		return c.body.Equal(i.body) && c.op.Equal(i.op)
	}
	return false
}

func (c Clause) String() string {
	ws := make([]string, len(c.pattern))
	for i, w := range c.pattern {
		ws[i] = w.String()
	}
	return fmt.Sprintf("clause(pattern: %s, body: %s)", strings.Join(ws, ", "), c.body)
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
	ok = i.cmd.Equal(j.cmd) && i.csq.Equal(j.csq)
	if i.alt != nil && j.alt != nil {
		ok = ok && i.alt.Equal(j.alt)
	}
	return ok
}

func (i If) String() string {
	if i.alt != nil {
		return fmt.Sprintf("if(cmd: %s, csq: %s, alt: %s)", i.cmd, i.csq, i.alt)
	}
	return fmt.Sprintf("if(cmd: %s, csq: %s)", i.cmd, i.csq)
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
	ok = u.cmd.Equal(i.cmd)
	if u.body != nil && i.body != nil {
		return ok && u.body.Equal(i.body)
	}
	return false
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
	ok = w.cmd.Equal(i.cmd)
	if w.body != nil && i.body != nil {
		return ok && w.body.Equal(i.body)
	}
	return false
}

func (w While) String() string {
	return fmt.Sprintf("while(cmd: %s, body: %s)", w.cmd, w.body)
}

type For struct {
	name  Token
	words []Word
	body  Command
}

func (f For) Execute() (int, error) {
	var (
		code int
		err  error
	)
	for i := range f.words {
		_ = i
		code, err = f.body.Execute()
	}
	return code, err
}

func (f For) Equal(other Command) bool {
	i, ok := other.(For)
	if !ok {
		return ok
	}
	if !f.name.Equal(i.name) {
		return false
	}
	if len(f.words) != len(i.words) {
		return false
	}
	for j, w := range f.words {
		if !w.Equal(i.words[j]) {
			return false
		}
	}
	if f.body != nil && i.body != nil {
		return f.body.Equal(i.body)
	}
	return false
}

func (f For) String() string {
	return fmt.Sprintf("for(words: %s, body: %s)", "", f.body.String())
}

type Break struct{}

func (_ Break) Execute() (int, error) {
	return 0, ErrBreak
}

func (_ Break) Equal(other Command) bool {
	_, ok := other.(Break)
	return ok
}

func (_ Break) String() string {
	return "break()"
}

type Continue struct{}

func (_ Continue) Execute() (int, error) {
	return 0, ErrContinue
}

func (_ Continue) Equal(other Command) bool {
	_, ok := other.(Continue)
	return ok
}

func (_ Continue) String() string {
	return "continue()"
}

type Assign struct {
	name Token
	word Word
}

func (a Assign) Execute() (int, error) {
	return 0, nil
}

func (a Assign) Equal(other Command) bool {
	i, ok := other.(Assign)
	if !ok {
		return ok
	}
	return a.name.Equal(i.name) && a.word.Equal(i.word)
}

func (a Assign) String() string {
	return fmt.Sprintf("assign(name: %s, word: %s)", a.name, a.word)
}
