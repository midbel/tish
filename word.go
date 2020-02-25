package tish

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Kind int

const (
	kindSimple Kind = iota
	kindSeq
	kindPipe
	kindPipeBoth
	kindAnd
	kindOr
	kindList
	kindSub
	kindExpr
	kindBraces
	kindWord
)

func (k Kind) String() string {
	switch k {
	case kindWord:
		return "word"
	case kindSimple:
		return "simple"
	case kindSeq:
		return "sequence"
	case kindPipe:
		return "pipe1"
	case kindPipeBoth:
		return "pipe2"
	case kindAnd:
		return "and"
	case kindOr:
		return "or"
	case kindList:
		return "list"
	case kindSub:
		return "substitution"
	case kindExpr:
		return "expression"
	case kindBraces:
		return "brace"
	default:
		return "unknown"
	}
}

type Word interface {
	Expand(*Env) ([]string, error)
	Equal(Word) bool
	fmt.Stringer

	asWord() Word
}

type Pipe struct {
	Word
	kind Kind
}

func (p Pipe) Equal(w Word) bool {
	other, ok := w.(Pipe)
	if !ok {
		return ok
	}
	return p.Word.Equal(other.Word)
}

func (p Pipe) String() string {
	var pipe string
	switch p.kind {
	case kindPipe, kindPipeBoth:
		pipe = p.kind.String()
	default:
		pipe = "pipeline"
	}
	return fmt.Sprintf("%s(%s)", pipe, p.Word)
}

type List struct {
	words []Word
	kind  Kind
}

func (i List) Expand(e *Env) ([]string, error) {
	ws := make([]string, 0, len(i.words)*4)
	for _, w := range i.words {
		xs, err := w.Expand(e)
		if err != nil {
			return nil, err
		}
		ws = append(ws, xs...)
	}
	if i.kind == kindWord {
		ws = []string{strings.Join(ws, "")}
	}
	return ws, nil
}

func (i List) String() string {
	var buf strings.Builder
	buf.WriteString(i.kind.String())
	buf.WriteRune(lparen)
	for i, w := range i.words {
		if i > 0 {
			buf.WriteRune(comma)
		}
		buf.WriteString(w.String())
	}
	buf.WriteRune(rparen)
	return buf.String()
}

func (i List) Equal(w Word) bool {
	other, ok := w.(List)
	if !ok || i.kind != other.kind || len(i.words) != len(other.words) {
		return false
	}
	for j := 0; j < len(i.words); j++ {
		if !i.words[j].Equal(other.words[j]) {
			return false
		}
	}
	return true
}

func (i List) asWord() Word {
	if i.kind == kindSub || i.kind == kindExpr || len(i.words) != 1 {
		return i
	}
	return i.words[0].asWord()
}

const (
	fdIn int = iota
	fdOut
	fdErr
	fdBoth

	modRead int = iota
	modWrite
	modAppend
	modRelink
)

type Redirect struct {
	file int
	mode int
	Word
}

func (r Redirect) Open(e *Env) (*os.File, error) {
	args, err := r.Expand(e)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("no words expanded")
	}
	var flag int
	switch r.mode {
	case modRead:
		flag = os.O_RDONLY
	case modWrite:
		flag = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	case modAppend:
		flag = os.O_CREATE | os.O_APPEND | os.O_WRONLY
	case modRelink:
	default:
		return nil, fmt.Errorf("unsupported mode")
	}
	return os.OpenFile(args[0], flag, 0644)
}

func (r Redirect) Equal(w Word) bool {
	other, ok := w.(Redirect)
	if !ok {
		return ok
	}
	if r.file == other.file && r.mode == other.mode {
		if r.Word == nil && other.Word == nil {
			return true
		}
		if r.Word != nil && other.Word == nil {
			return false
		}
		if r.Word == nil && other.Word != nil {
			return false
		}
		return r.Word.Equal(other.Word)
	}
	return false
}

func (r Redirect) String() string {
	var str string
	switch r.mode {
	case modRead:
		str = "read"
	case modWrite:
		str = "write"
	case modAppend:
		str = "append"
	case modRelink:
		str = "relink"
	}
	if r.Word == nil {
		return str
	}
	return fmt.Sprintf("%s(%s)", str, r.Word.String())
}

func (r Redirect) asWord() Word {
	return r
}

type Brace struct {
	prolog Word
	epilog Word
	word   Word
}

func (b Brace) Expand(e *Env) ([]string, error) {
	if b.word == nil {
		return nil, nil
	}
	ws, err := b.word.Expand(e)
	if err != nil {
		return nil, err
	}
	for i, w := range []Word{b.prolog, b.epilog} {
		if w == nil {
			continue
		}
		xs, err := w.Expand(e)
		if err != nil {
			return nil, err
		}
		ws = combineWords(ws, xs, i == 0)
	}
	return ws, nil
}

func (b Brace) String() string {
	var buf strings.Builder
	buf.WriteString("brace(")
	if b.prolog != nil {
		buf.WriteString("pre(")
		buf.WriteString(b.prolog.String())
		buf.WriteString(")")
	}

	buf.WriteRune(lcurly)
	buf.WriteString(b.word.String())
	buf.WriteRune(rcurly)

	if b.epilog != nil {
		buf.WriteString("post(")
		buf.WriteString(b.epilog.String())
		buf.WriteString(")")
	}
	buf.WriteString(")")
	return buf.String()
}

func (b Brace) Equal(w Word) bool {
	_, ok := w.(Brace)
	return ok
}

func (b Brace) asWord() Word {
	return b
}

type Variable struct {
	ident  string
	quoted bool
	apply  Apply
}

func (v Variable) Expand(e *Env) ([]string, error) {
	if v.apply == nil {
		return e.Get(v.ident)
	}
	return v.apply.Apply(v.ident, e)
}

func (v Variable) String() string {
	return fmt.Sprintf("variable(%s)", v.ident)
}

func (v Variable) Equal(w Word) bool {
	other, ok := w.(Variable)
	if !ok {
		return false
	}
	return other.ident == v.ident && other.quoted == v.quoted // && other.apply.Equal(v.apply)
}

func (v Variable) Eval(e *Env) (Number, error) {
	vs, err := e.Get(v.ident)
	if err != nil {
		return 0, err
	}
	x, err := strconv.ParseInt(vs[0], 10, 64)
	if err != nil {
		return 0, err
	}
	return Number(x), nil
}

func (v Variable) asWord() Word {
	return v
}

type Literal string

func (i Literal) Expand(_ *Env) ([]string, error) {
	return []string{string(i)}, nil
}

func (i Literal) String() string {
	return fmt.Sprintf("literal(%s)", string(i))
}

func (i Literal) Equal(w Word) bool {
	other, ok := w.(Literal)
	if !ok {
		return false
	}
	return string(other) == string(i)
}

func (i Literal) asWord() Word {
	return i
}

type Number int64

func (n Number) Eval(_ *Env) (Number, error) {
	return n, nil
}

func (n Number) Expand(_ *Env) ([]string, error) {
	x := strconv.FormatInt(int64(n), 10)
	return []string{x}, nil
}

func (n Number) String() string {
	return fmt.Sprintf("number(%d)", int64(n))
}

func (n Number) Equal(w Word) bool {
	other, ok := w.(Number)
	if !ok {
		return ok
	}
	return int64(n) == int64(other)
}

func (n Number) asWord() Word {
	return n
}

type Assignment struct {
	ident string
	word  Word
}

func (a Assignment) Expand(e *Env) ([]string, error) {
	vs, err := a.word.Expand(e)
	if err != nil {
		return nil, err
	}
	e.Set(a.ident, vs)
	return nil, nil
}

func (a Assignment) String() string {
	if a.word == nil {
		return fmt.Sprintf("assignment(%s)", a.ident)
	}
	return fmt.Sprintf("assignment(%s=%s)", a.ident, a.word.String())
}

func (a Assignment) Equal(w Word) bool {
	other, ok := w.(Assignment)
	if !ok {
		return ok
	}
	if a.ident != other.ident {
		return false
	}
	if a.word == nil && other.word == nil {
		return true
	}
	if a.word == nil || other.word == nil {
		return false
	}
	return a.word.Equal(other.word)
}

func (a Assignment) asWord() Word {
	return a
}

func combineWords(ws, ps []string, prefix bool) []string {
	var words []string
	for i := range ws {
		for j := range ps {
			if prefix {
				words = append(words, ps[j]+ws[i])
			} else {
				words = append(words, ws[i]+ps[j])
			}
		}
	}
	if len(words) == 0 {
		words = ws
	}
	return words
}

type If struct {
	expr Expr
	csq  Word
	alt  Word
}

func (i If) Expand(e *Env) ([]string, error) {
	return nil, nil
}

func (i If) Equal(w Word) bool {
	other, ok := w.(If)
	if !ok {
		return ok
	}
	return i.expr.Equal(other.expr) && i.csq.Equal(other.csq) && i.alt.Equal(other.alt)
}

func (i If) String() string {
	return "if"
}

func (i If) asWord() Word {
	return i
}

type For struct {
	expr Expr
	word Word
}

func (f For) Expand(e *Env) ([]string, error) {
	return nil, nil
}

func (f For) Equal(w Word) bool {
	other, ok := w.(For)
	if !ok {
		return ok
	}
	return f.expr.Equal(other.expr) && f.word.Equal(other.expr)
}

func (f For) asWord() Word {
	return f
}

func (f For) String() string {
	return "for"
}

type Match struct {
	expr  Expr
	words []Word
}

func (m Match) Expand(e *Env) ([]string, error) {
	return nil, nil
}

func (m Match) Equal(w Word) bool {
	other, ok := w.(Match)
	if !ok {
		return ok
	}
	if !m.expr.Equal(other.expr) {
		return false
	}
	if len(m.words) != len(other.words) {
		return false
	}
	for i, w := range m.words {
		if !w.Equal(other.words[i]) {
			return false
		}
	}
	return true
}

func (m Match) String() string {
	return "match"
}

func (m Match) asWord() Word {
	return m
}
