package token

import (
	"sort"
)

const (
	KwFor      = "for"
	KwDo       = "do"
	KwDone     = "done"
	KwIn       = "in"
	KwWhile    = "while"
	KwUntil    = "until"
	KwIf       = "if"
	KwFi       = "fi"
	KwThen     = "then"
	KwElse     = "else"
	KwCase     = "case"
	KwEsac     = "esac"
	KwBreak    = "break"
	KwContinue = "continue"
)

var list = []string{
	KwFor,
	KwDo,
	KwDone,
	KwIn,
	KwWhile,
	KwUntil,
	KwIf,
	KwFi,
	KwThen,
	KwElse,
	KwCase,
	KwEsac,
	KwBreak,
	KwContinue,
}

func init() {
	sort.Strings(list)
}

func IsKeyword(str string) bool {
	i := sort.SearchStrings(list, str)
	return i < len(list) && list[i] == str
}
