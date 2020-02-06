package tish

import (
	"fmt"
)

const (
	space      = ' '
	tab        = '\t'
	squote     = '\''
	dquote     = '"'
	backslash  = '\\'
	dollar     = '$'
	semicolon  = ';'
	pipe       = '|'
	ampersand  = '&'
	equal      = '='
	newline    = '\n'
	lparen     = '('
	rparen     = ')'
	lcurly     = '{'
	rcurly     = '}'
	underscore = '_'
	pound      = '#'
	plus       = '+'
	minus      = '-'
	div        = '/'
	mul        = '*'
	modulo     = '%'
	dot        = '.'
	comma      = ','
	tilde      = '~'
)

const (
	tokEOF rune = -(iota + 1)
	tokBlank
	tokQuoted
	tokWord
	tokInteger
	tokFloat
	tokVar
	tokComment
	tokIllegal
	tokError
	tokBeginSub
	tokEndSub
	tokBeginArith
	tokEndArith
	tokBeginBrace
	tokEndBrace
	tokBeginList
	tokEndList
	tokSequence
	tokAnd
	tokOr
	tokBackground
)

var (
	eof   = Token{Type: tokEOF}
	blank = Token{Type: tokBlank}
)

type Token struct {
	Literal string
	Type    rune
}

func (t Token) Equal(other Token) bool {
	return t.Type == other.Type && t.Literal == other.Literal
}

func (t Token) String() string {
	var str string
	switch t.Type {
	case tokError:
		return "<error>"
	case tokBlank:
		return "<blank>"
	case tokEOF:
		return "eof"
	case tokWord:
		str = "word"
	case tokVar:
		str = "var"
	case tokInteger:
		str = "integer"
	case tokFloat:
		str = "float"
	case tokComment:
		str = "comment"
	case tokAnd:
		return "<and>"
	case tokOr:
		return "<or>"
	case tokBeginSub, tokBeginBrace, tokBeginArith, tokBeginList:
		return "<begin>"
	case tokEndSub, tokEndBrace, tokEndArith, tokEndList:
		return "<end>"
	case pipe:
		return "<pipe>"
	case semicolon:
		return "<semicolon>"
	case ampersand:
		return "<ampersand>"
	case plus:
		return "<add>"
	case minus:
		return "<subtract>"
	case mul:
		return "<multiply>"
	case div:
		return "<divide>"
	case modulo:
		return "<modulo>"
	case lparen, rparen, lcurly, rcurly:
		return "<group>"
	case tilde:
		return "<tilde>"
	case equal:
		return "<assign>"
	default:
		return fmt.Sprintf("unknown(%d)", t.Type)
	}
	return fmt.Sprintf("%s(%s)", str, t.Literal)
}
