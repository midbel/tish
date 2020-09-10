package tish

import (
	"fmt"
	"strings"
)

type Command interface {
	fmt.Stringer
}

type Word struct {
	tokens []Token
}

func (w Word) Expand(env *Env) string {
	ws := make([]string, 0, len(w.tokens))
	for _, tok := range w.tokens {
		var str string
		switch tok.Type {
		case TokLiteral:
			str = tok.Literal
		case TokVariable:
			word := env.Resolve(tok.Literal)
			if !word.IsZero() {
				str = word.Expand(env)
			}
		default:
			continue
		}
		ws = append(ws, str)
	}
	return strings.Join(ws, "")
}

func (w Word) String() string {
	ws := make([]string, len(w.tokens))
	for i := range w.tokens {
		ws[i] = w.tokens[i].Literal
	}
	return fmt.Sprintf("word(%s)", strings.Join(ws, ""))
}

func (w Word) IsZero() bool {
	return len(w.tokens) == 0
}

type Simple struct {
	env   []Assign
	words []Word
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

func (a And) String() string {
	return fmt.Sprintf("and(%s, %s)", a.left, a.right)
}

type Or struct {
	left  Command
	right Command
}

func (o Or) String() string {
	return fmt.Sprintf("or(%s, %s)", o.left, o.right)
}

type Case struct {
	word    Word
	clauses []Clause
}

func (c Case) String() string {
	ws := make([]string, len(c.clauses))
	for i, c := range c.clauses {
		ws[i] = c.String()
	}
	return fmt.Sprintf("case(word: %s, body: %s)", c.word, ws)
}

type Clause struct {
	pattern []Word
	body    Command
	op      Token
}

func (c Clause) Match(str string, env *Env) bool {
	for _, w := range c.pattern {
		if str == w.Expand(env) {
			return true
		}
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

func (u Until) String() string {
	return fmt.Sprintf("until(cmd: %s, body: %s)", u.cmd, u.body)
}

type While struct {
	cmd  Command
	body Command
}

func (w While) String() string {
	return fmt.Sprintf("while(cmd: %s, body: %s)", w.cmd, w.body)
}

type For struct {
	ident Token
	words []Word
	body  Command
}

func (f For) String() string {
	return fmt.Sprintf("for(words: %s, body: %s)", "", f.body.String())
}

type Break struct{}

func (_ Break) String() string {
	return "break()"
}

type Continue struct{}

func (_ Continue) String() string {
	return "continue()"
}

type Assign struct {
	ident Token
	word  Word
}

func (a Assign) String() string {
	return fmt.Sprintf("assign(ident: %s, word: %s)", a.ident, a.word)
}
