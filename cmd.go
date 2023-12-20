package tish

import (
	"fmt"
)

type Command interface {
	Expand(Environment) ([]string, error)
}

type cmdSingle struct {
	export   []Command
	words    []Word
	redirect []Word
}

func single(words []Word, export []Command) Command {
	return cmdSingle{
		export: export,
		words:  words,
	}
}

func (c cmdSingle) Expand(env Environment) ([]string, error) {
	var list []string
	for _, w := range c.words {
		str, err := w.Expand(env)
		if err != nil {
			return nil, err
		}
		list = append(list, str...)
	}
	return list, nil
}

type cmdAnd struct {
	left  Command
	right Command
}

func andCommand(left, right Command) Command {
	return cmdAnd{
		left:  left,
		right: right,
	}
}

func (_ cmdAnd) Expand(_ Environment) ([]string, error) {
	return nil, unexpandableCmd("and")
}

type cmdOr struct {
	left  Command
	right Command
}

func orCommand(left, right Command) Command {
	return cmdOr{
		left:  left,
		right: right,
	}
}

func (_ cmdOr) Expand(_ Environment) ([]string, error) {
	return nil, unexpandableCmd("or")
}

type cmdPipe struct {
	list []Command
}

func pipeCommand(list []Command) Command {
	return cmdPipe{
		list: list,
	}
}

func (_ cmdPipe) Expand(_ Environment) ([]string, error) {
	return nil, unexpandableCmd("pipe")
}

type cmdAssign struct {
	ident string
	word  Word
}

func assignCommand(ident string, word Word) Command {
	return cmdAssign{
		ident: ident,
		word:  word,
	}
}

func (c cmdAssign) Expand(env Environment) ([]string, error) {
	list, err := c.word.Expand(env)
	if err != nil {
		return nil, err
	}
	env.Define(c.ident, list)
	return nil, nil
}

type cmdGroup struct {
	commands []Command
}

func groupCommand(list []Command) Command {
	return cmdGroup{
		commands: list,
	}
}

func (_ cmdGroup) Expand(_ Environment) ([]string, error) {
	return nil, unexpandableCmd("group")
}

type cmdList struct {
	commands []Command
}

func listCommand(list []Command) Command {
	if len(list) == 1 {
		return list[0]
	}
	return cmdList{
		commands: list,
	}
}

func (_ cmdList) Expand(_ Environment) ([]string, error) {
	return nil, unexpandableCmd("list")
}

type cmdTest struct {
	Expr
}

func testCommand(e Expr) Command {
	return cmdTest{
		Expr: e,
	}
}

func (c cmdTest) Expand(env Environment) ([]string, error) {
	ok, err := c.Expr.Test(env)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, nil
	}
	return nil, ErrFalse
}

type cmdIf struct {
	test Command
	csq  Command
	alt  Command
}

func (_ cmdIf) Expand(_ Environment) ([]string, error) {
	return nil, unexpandableCmd("if")
}

type cmdFor struct {
	ident string
	iter  Command
	body  Command
}

func (_ cmdFor) Expand(_ Environment) ([]string, error) {
	return nil, unexpandableCmd("for")
}

type cmdWhile struct {
	iter Command
	body Command
}

func (_ cmdWhile) Expand(_ Environment) ([]string, error) {
	return nil, unexpandableCmd("while")
}

type cmdUntil struct {
	iter Command
	body Command
}

func (_ cmdUntil) Expand(_ Environment) ([]string, error) {
	return nil, unexpandableCmd("until")
}

func unexpandableCmd(ident string) error {
	return fmt.Errorf("%s can not be expanded", ident)
}
