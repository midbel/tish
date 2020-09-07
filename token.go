package tish

import (
	"fmt"
)

type Kind int

const (
	TokEOF Kind = -(iota + 1)
	TokBlank
	TokKeyword
	TokLiteral
	TokVariable
	TokComment
	TokInvalid
	TokSemicolon
	TokAnd
	TokOr
	TokPipe
	TokBackground
	TokAssign
	TokEqual
	TokNotEqual
	TokBreak
	TokContinue
	TokFallthrough
	TokBegGroup
	TokEndGroup
)

func (k Kind) IsBreak() bool {
	return k == TokBreak || k == TokContinue || k == TokFallthrough
}

func (k Kind) EndOfWord() bool {
	return k == TokBlank || k == TokAnd ||
		k == TokOr || k == TokSemicolon ||
		k == TokPipe || k == TokBackground ||
		k == TokEndGroup || k == TokBegGroup
}

func (k Kind) EndOfCommand() bool {
	return k == TokSemicolon || k == TokEOF
}

func (k Kind) String() string {
	var str string
	switch k {
	case TokEOF:
		str = "eof"
	case TokBlank:
		str = "blank"
	case TokLiteral:
		str = "literal"
	case TokKeyword:
		str = "keyword"
	case TokVariable:
		str = "variable"
	case TokComment:
		str = "comment"
	case TokInvalid:
		str = "invalid"
	case TokSemicolon:
		str = "semicolon"
	case TokAnd:
		str = "and"
	case TokOr:
		str = "or"
	case TokPipe:
		str = "pipe"
	case TokBackground:
		str = "background"
	case TokAssign:
		str = "assign"
	case TokBreak:
		str = "break"
	case TokContinue:
		str = "continue"
	case TokFallthrough:
		str = "fallthrough"
	case TokBegGroup, TokEndGroup:
		str = "group"
	default:
		str = "unknown"
	}
	return str
}

type Token struct {
	Literal string
	Type    Kind
	Quoted  bool
}

func (t Token) Equal(other Token) bool {
	return t.Type == other.Type && t.Literal == other.Literal && t.Quoted == other.Quoted
}

func (t Token) String() string {
	switch t.Type {
	case TokLiteral, TokComment, TokInvalid, TokVariable, TokKeyword:
		return fmt.Sprintf("<%s(%s)>", t.Type, t.Literal)
	default:
		return fmt.Sprintf("<%s>", t.Type)
	}
}
