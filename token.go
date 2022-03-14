package tish

import (
	"fmt"
)

const (
	EOF = -(iota + 1)
	Keyword
	Blank
	Literal
	Numeric
	Quote
	Comment
	Variable
	Comma
	BegExp
	EndExp
	BegBrace
	EndBrace
	Range
	Seq
	List
	Pipe
	PipeBoth
	BegMath
	EndMath
	Not
	And
	Or
	Cond
	Alt
	Eq
	Ne
	Lt
	Le
	Gt
	Ge
	Add
	Sub
	Mod
	Div
	Mul
	Pow
	Inc
	Dec
	LeftShift
	RightShift
	BitAnd
	BitOr
	BitNot
	BitXor
	BegSub
	EndSub
	Assign
	RedirectIn   // < | 0<
	RedirectOut  // > | 1>
	RedirectErr  // 2>
	RedirectBoth // &>
	AppendOut    // >> | 1>>
	AppendErr    // 2>>
	AppendBoth   // &>>
	BegTest      // [[
	EndTest      // ]]
	StrEmpty
	StrNotEmpty
	SameFile
	OlderThan
	NewerThan
	FileExists
	FileLink
	FileDir
	FileExec
	FileRegular
	FileRead
	FileWrite
	FileSize
	Length         // ${#var}
	Slice          // ${var:from:to}
	Replace        // ${var/from/to}
	ReplaceAll     // ${var//from/to}
	ReplaceSuffix  // ${var/%from/to}
	ReplacePrefix  // ${var/#from/to}
	TrimSuffix     // ${var%suffix}
	TrimSuffixLong // ${var%%suffix}
	TrimPrefix     // ${var#suffix}
	TrimPrefixLong // ${var##suffix}
	Lower          // ${var,}
	LowerAll       // ${var,,}
	Upper          // ${var^}
	UpperAll       // ${var^^}
	PadLeft        // ${var:<10:0}
	PadRight       // ${var:>10:0}
	ValIfUnset     // ${var:-val}
	SetValIfUnset  // ${var:=val}
	ValIfSet       // ${var:+val}
	ExitIfUnset    // ${var:?val}
	Invalid
)

var colonOps = map[rune]rune{
	minus:    ValIfUnset,
	plus:     ValIfSet,
	equal:    SetValIfUnset,
	question: ExitIfUnset,
	langle:   PadLeft,
	rangle:   PadRight,
}

var slashOps = map[rune]rune{
	slash:   ReplaceAll,
	percent: ReplaceSuffix,
	pound:   ReplacePrefix,
}

const (
	kwFor      = "for"
	kwDo       = "do"
	kwDone     = "done"
	kwIn       = "in"
	kwWhile    = "while"
	kwUntil    = "until"
	kwIf       = "if"
	kwFi       = "fi"
	kwThen     = "then"
	kwElse     = "else"
	kwCase     = "case"
	kwEsac     = "esac"
	kwBreak    = "break"
	kwContinue = "continue"
)

type Token struct {
	Literal string
	Type    rune
}

func (t Token) IsSequence() bool {
	switch t.Type {
	case And, Or, List, Pipe, PipeBoth, Comment, EndSub, Comma:
		return true
	default:
		if t.IsRedirect() {
			return true
		}
		return false
	}
}

func (t Token) IsList() bool {
	return t.Type == Range || t.Type == Seq
}

func (t Token) IsRedirect() bool {
	switch t.Type {
	case RedirectIn, RedirectOut, RedirectErr, RedirectBoth, AppendOut, AppendErr, AppendBoth:
		return true
	default:
		return false
	}
}

func (t Token) Eow() bool {
	return t.Type == EndSub || t.Type == Blank || t.IsSequence() || t.IsRedirect() || t.IsEOF() || t.IsComment()
}

func (t Token) IsEOF() bool {
	return t.Type == EOF
}

func (t Token) IsComment() bool {
	return t.Type == Comment
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	case EOF:
		return "<eof>"
	case Comma:
		return "<comma>"
	case Blank:
		return "<blank>"
	case Quote:
		return "<quote>"
	case Eq:
		return "<eq>"
	case Ne:
		return "<ne>"
	case And:
		return "<and>"
	case Or:
		return "<or>"
	case Pipe:
		return "<pipe>"
	case PipeBoth:
		return "<pipe-both>"
	case BegSub:
		return "<beg-sub>"
	case EndSub:
		return "<end-sub>"
	case List:
		return "<list>"
	case BegExp:
		return "<beg-expansion>"
	case EndExp:
		return "<end-expansion>"
	case BegMath:
		return "<beg-arithmetic>"
	case EndMath:
		return "<end-arithmetic>"
	case Cond:
		return "<ternary>"
	case Alt:
		return "<ternary-alt>"
	case Not:
		return "<not>"
	case Add:
		return "<add>"
	case Sub:
		return "<sub>"
	case Mod:
		return "<mod>"
	case Div:
		return "<div>"
	case Mul:
		return "<mul>"
	case Pow:
		return "<pow>"
	case Inc:
		return "<inc>"
	case Dec:
		return "<dec>"
	case LeftShift:
		return "<left-shift>"
	case RightShift:
		return "<right-shift>"
	case BitAnd:
		return "<bit-and>"
	case BitOr:
		return "<bit-or>"
	case BitNot:
		return "<bit-not>"
	case BegBrace:
		return "<beg-brace>"
	case EndBrace:
		return "<end-brace>"
	case Range:
		return "<range>"
	case Seq:
		return "<sequence>"
	case Length:
		return "<length>"
	case Slice:
		return "<slice>"
	case Replace:
		return "<replace>"
	case ReplaceAll:
		return "<replace-all>"
	case ReplaceSuffix:
		return "<replace-suffix>"
	case ReplacePrefix:
		return "<replace-prefix>"
	case TrimSuffix:
		return "<trim-suffix>"
	case TrimSuffixLong:
		return "<trim-suffix-long>"
	case TrimPrefix:
		return "<trim-prefix>"
	case TrimPrefixLong:
		return "<trim-prefix-long>"
	case Lower:
		return "<lower>"
	case LowerAll:
		return "<lower-all>"
	case Upper:
		return "<upper>"
	case UpperAll:
		return "<upper-all>"
	case PadLeft:
		return "<padding-left>"
	case PadRight:
		return "<padding-right>"
	case ValIfUnset:
		return "<val-if-unset>"
	case SetValIfUnset:
		return "<set-val-if-unset>"
	case ValIfSet:
		return "<val-if-set>"
	case ExitIfUnset:
		return "<exit-if-unset>"
	case Assign:
		return "<assignment>"
	case RedirectIn:
		return "<redirect-in>"
	case RedirectOut:
		return "<redirect-out>"
	case RedirectErr:
		return "<redirect-err>"
	case RedirectBoth:
		return "<redirect-both>"
	case AppendOut:
		return "<append-out>"
	case AppendErr:
		return "<append-err>"
	case AppendBoth:
		return "<append-Both>"
	case BegTest:
		return "<beg-test>"
	case EndTest:
		return "<end-test>"
	case Variable:
		prefix = "variable"
	case Comment:
		prefix = "comment"
	case Literal:
		prefix = "literal"
	case Numeric:
		prefix = "numeric"
	case Invalid:
		prefix = "invalid"
	case Keyword:
		prefix = "keyword"
	default:
		prefix = "unknown"
	}
	return fmt.Sprintf("%s(%s)", prefix, t.Literal)
}
