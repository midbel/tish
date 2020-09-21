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
	TokNumber
	TokVariable
	TokComment
	TokInvalid
	TokSemicolon
	TokAnd
	TokOr
	TokPipe
	TokBackground
	TokAssign
	TokBreak
	TokContinue
	TokFallthrough
	TokBegGroup
	TokEndGroup
	TokBegTest
	TokEndTest
	TokBegArith
	TokEndArith
	TokAdd
	TokSub
	TokMul
	TokDiv
	TokMod
	TokLeftShift
	TokRightShift
	TokGreater
	TokGreateq
	TokLesser
	TokLesseq
	TokEqual
	TokNotEqual
	TokNot
	TokBegExp
	TokEndExp
	TokLen
	TokTrimSuffix
	TokTrimSuffixLong
	TokTrimPrefix
	TokTrimPrefixLong
	TokReplace
	TokReplaceAll
	TokReplacePrefix
	TokReplaceSuffix
	TokLower
	TokLowerAll
	TokUpper
	TokUpperAll
	TokReverse
	TokReverseAll
	TokSlice
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
	case TokNumber:
		str = "number"
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
	case TokBegTest, TokEndTest:
		str = "test"
	case TokBegArith, TokEndArith:
		str = "arithmetic"
	case TokBegExp, TokEndExp:
		str = "expansion"
	case TokAdd:
		str = "add"
	case TokSub:
		str = "subtract"
	case TokMul:
		str = "multiply"
	case TokDiv:
		str = "divide"
	case TokMod:
		str = "modulo"
	case TokSlice:
		str = "slice"
	case TokReplace, TokReplaceAll, TokReplacePrefix, TokReplaceSuffix:
		str = "replace"
	case TokTrimSuffix, TokTrimSuffixLong, TokTrimPrefix, TokTrimPrefixLong:
		str = "trim"
	case TokLower, TokLowerAll, TokUpper, TokUpperAll, TokReverse, TokReverseAll:
		str = "transform"
	case TokLen:
		str = "length"
	default:
		str = "unknown"
	}
	return str
}

type Position struct {
	Line int
	Col  int
}

type Token struct {
	Literal string
	Type    Kind
	Quoted  bool
	Position
}

func Compare(fst, snd Token) bool {
	return fst.Equal(snd)
}

func (t Token) Equal(other Token) bool {
	return t.Type == other.Type && t.Literal == other.Literal && t.Quoted == other.Quoted
}

func (t Token) String() string {
	switch t.Type {
	case TokLiteral, TokNumber, TokComment, TokInvalid, TokVariable, TokKeyword:
		return fmt.Sprintf("<%s(%s)>", t.Type, t.Literal)
	default:
		return fmt.Sprintf("<%s>", t.Type)
	}
}

func (t Token) isKeyword() bool {
	return t.Type == TokKeyword
}

func (t Token) isSimple() bool {
	switch t.Type {
	case TokLiteral, TokVariable, TokBegArith, TokBegExp:
		return true
	default:
		return false
	}
}

func (t Token) isTrim() bool {
	switch t.Type {
	case TokTrimSuffix, TokTrimSuffixLong, TokTrimPrefix, TokTrimPrefixLong:
		return true
	default:
		return false
	}
}

func (t Token) isReplace() bool {
	switch t.Type {
	case TokReplace, TokReplaceAll, TokReplaceSuffix, TokReplacePrefix:
		return true
	default:
		return false
	}
}

func (t Token) isTransform() bool {
	switch t.Type {
	case TokLower, TokLowerAll, TokUpper, TokUpperAll, TokReverse, TokReverseAll:
		return true
	default:
		return false
	}
}

func (t Token) isSlice() bool {
	return t.Type == TokSlice
}
