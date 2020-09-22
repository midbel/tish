package tish

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Word interface {
	Command
	Expand(Environment) string
}

type Evaluator interface {
	fmt.Stringer
	Eval(Environment) (int, error)
	Equal(Evaluator) bool
}

type StringList interface {
	Fields() []string
	Split(string) []string
}

type Unquoted string

func (u Unquoted) Fields() []string {
	return u.Split(" \t\n")
}

func (u Unquoted) Split(cutset string) []string {
	return splitString(string(u), []rune(cutset))
}

type Quoted string

func (q Quoted) Fields() []string {
	return q.Split("")
}

func (q Quoted) Split(cutset string) []string {
	return splitString(string(q), []rune(cutset))
}

func splitString(str string, cutset []rune) []string {
	if len(cutset) == 0 || str == "" {
		return []string{str}
	}
	sort.Slice(cutset, func(i, j int) bool {
		return cutset[i] < cutset[j]
	})
	return strings.FieldsFunc(str, func(r rune) bool {
		x := sort.Search(len(cutset), func(i int) bool {
			return cutset[i] >= r
		})
		return x < len(cutset) && cutset[x] == r
	})
}

type WordList struct {
	words []Word
}

func (w WordList) Expand(env Environment) string {
	ws := make([]string, 0, len(w.words))
	for _, w := range w.words {
		str := w.Expand(env)
		ws = append(ws, str)
	}
	return strings.Join(ws, "")
}

func (w WordList) String() string {
	ws := make([]string, len(w.words))
	for i := range w.words {
		ws[i] = w.words[i].String()
	}
	return fmt.Sprintf("wordlist(%s)", strings.Join(ws, ", "))
}

func (w WordList) Equal(other Command) bool {
	c, ok := other.(WordList)
	if !ok {
		return ok
	}
	if len(w.words) != len(c.words) {
		return false
	}
	for i := range w.words {
		if !w.words[i].Equal(c.words[i]) {
			return false
		}
	}
	return true
}

func (w WordList) asWord() Word {
	if len(w.words) == 1 {
		return w.words[0]
	}
	return w
}

type Literal struct {
	token Token
}

func (i Literal) Expand(env Environment) string {
	var str string
	switch i.token.Type {
	case TokLiteral:
		str = i.token.Literal
	case TokVariable:
		str = env.Resolve(i.token.Literal)
	default:
	}
	return str
}

func (i Literal) String() string {
	return i.token.String()
}

func (i Literal) Equal(other Command) bool {
	c, ok := other.(Literal)
	if !ok {
		return ok
	}
	return Compare(i.token, c.token)
}

type Slice struct {
	ident  Token
	offset Token
	length Token
}

func (s Slice) String() string {
	return fmt.Sprintf("slice(ident: %s, offset: %s, length: %s)", s.ident, s.offset, s.length)
}

func (s Slice) Equal(other Command) bool {
	c, ok := other.(Slice)
	if !ok {
		return ok
	}
	return Compare(s.ident, c.ident) && Compare(s.offset, c.offset) && Compare(s.length, c.length)
}

func (s Slice) Expand(env Environment) string {
	str := env.Resolve(s.ident.Literal)
	if str == "" {
		return str
	}
	offset, length := s.convert()

	if offset < 0 {
		offset = len(str) + offset
	}
	if length < 0 {
		length = len(str) + length
	}
	switch {
	case offset == 0 && length == 0:
		return str
	case offset > 0 && length == 0:
		if offset >= len(str) {
			return ""
		}
		return str[offset:]
	case offset == 0 && length > 0:
		if length >= len(str) {
			return str
		}
		return str[:length]
	default:
		return str[offset : offset+length]
	}
}

func (s Slice) convert() (int, int) {
	offset, _ := strconv.Atoi(s.offset.Literal)
	length, _ := strconv.Atoi(s.length.Literal)
	return offset, length
}

type Trim struct {
	ident Token
	str   Token
	part  Token
}

func (t Trim) String() string {
	return fmt.Sprintf("trim(ident: %s, str: %s, part: %s)", t.ident, t.str, t.part)
}

func (t Trim) Equal(other Command) bool {
	c, ok := other.(Trim)
	if !ok {
		return ok
	}
	return Compare(t.ident, c.ident) && Compare(t.str, c.str) && Compare(t.part, c.part)
}

func (t Trim) Expand(env Environment) string {
	str := env.Resolve(t.ident.Literal)
	if str == "" {
		return str
	}
	switch t.part.Type {
	case TokTrimSuffix:
		str = t.trimSuffix(str, t.str.Literal, true)
	case TokTrimSuffixLong:
		str = t.trimSuffix(str, t.str.Literal, false)
	case TokTrimPrefix:
		str = t.trimPrefix(str, t.str.Literal, true)
	case TokTrimPrefixLong:
		str = t.trimPrefix(str, t.str.Literal, false)
	}
	return str
}

func (t Trim) trimSuffix(str, suffix string, short bool) string {
	for strings.HasSuffix(str, suffix) {
		str = strings.TrimSuffix(str, suffix)
		if short {
			break
		}
	}
	return str
}

func (t Trim) trimPrefix(str, prefix string, short bool) string {
	if short {
	}
	for strings.HasPrefix(str, prefix) {
		str = strings.TrimPrefix(str, prefix)
		if short {
			break
		}
	}
	return str
}

type Replace struct {
	ident Token
	src   Token
	dst   Token
	op    Token
}

func (r Replace) String() string {
	return fmt.Sprintf("replace(ident: %s, src: %s, dst: %s, op: %s)", r.ident, r.src, r.dst, r.op)
}

func (r Replace) Equal(other Command) bool {
	c, ok := other.(Replace)
	if !ok {
		return ok
	}
	return Compare(r.ident, c.ident) && Compare(r.src, c.src) && Compare(r.dst, c.dst) && Compare(r.op, c.op)
}

func (r Replace) Expand(env Environment) string {
	str := env.Resolve(r.ident.Literal)
	if str == "" {
		return str
	}
	switch r.op.Type {
	case TokReplace:
		str = strings.Replace(str, r.src.Literal, r.dst.Literal, 1)
	case TokReplaceAll:
		str = strings.ReplaceAll(str, r.src.Literal, r.dst.Literal)
	case TokReplaceSuffix:
		if strings.HasSuffix(str, r.src.Literal) {
			str = strings.TrimSuffix(str, r.src.Literal)
			str += r.dst.Literal
		}
	case TokReplacePrefix:
		if strings.HasPrefix(str, r.src.Literal) {
			str = strings.TrimPrefix(str, r.src.Literal)
			str = r.dst.Literal + str
		}
	}
	return str
}

type Transform struct {
	ident   Token
	pattern Token
	op      Token
}

func (t Transform) String() string {
	return fmt.Sprintf("case(ident: %s, case: %s)", t.ident, t.op)
}

func (t Transform) Equal(other Command) bool {
	c, ok := other.(Transform)
	if !ok {
		return ok
	}
	return Compare(t.ident, c.ident) && Compare(t.op, c.op)
}

func (t Transform) Expand(env Environment) string {
	str := env.Resolve(t.ident.Literal)
	if str == "" {
		return str
	}
	switch t.op.Type {
	case TokLower:
		str = t.toLowerCase(str, false)
	case TokLowerAll:
		str = t.toLowerCase(str, true)
	case TokUpper:
		str = t.toUpperCase(str, false)
	case TokUpperAll:
		str = t.toUpperCase(str, true)
	case TokReverse:
		str = t.reverseCase(str, false)
	case TokReverseAll:
		str = t.reverseCase(str, true)
	}
	return str
}

func (t Transform) toUpperCase(str string, all bool) string {
	if !all {
		rs := []rune(str)
		rs[0] = unicode.ToUpper(rs[0])
		return string(rs)
	}
	return strings.ToUpper(str)
}

func (t Transform) toLowerCase(str string, all bool) string {
	if !all {
		rs := []rune(str)
		rs[0] = unicode.ToLower(rs[0])
		return string(rs)
	}
	return strings.ToLower(str)
}

func (t Transform) reverseCase(str string, all bool) string {
	rs := []rune(str)
	for i := 0; i < len(rs); i++ {
		if i > 0 && !all {
			break
		}
		if !unicode.IsLetter(rs[i]) {
			continue
		}
		if unicode.IsUpper(rs[i]) {
			rs[i] = unicode.ToLower(rs[i])
		} else {
			rs[i] = unicode.ToUpper(rs[i])
		}
	}
	return string(rs)
}

type Length struct {
	ident Token
}

func (e Length) String() string {
	return fmt.Sprintf("length(ident: %s)", e.ident)
}

func (e Length) Equal(other Command) bool {
	c, ok := other.(Length)
	if !ok {
		return ok
	}
	return Compare(e.ident, c.ident)
}

func (e Length) Expand(env Environment) string {
	str := env.Resolve(e.ident.Literal)
	return strconv.Itoa(len(str))
}

type Serie struct {
	tokens []Token
}

type Range struct {
	first Token
	last  Token
}

type Expr struct {
	eval Evaluator
}

func (e Expr) Expand(env Environment) string {
	r, err := e.eval.Eval(env)
	if err != nil {
		return ""
	}
	return strconv.Itoa(r)
}

func (e Expr) String() string {
	return fmt.Sprintf("expr(%s)", e.eval)
}

func (e Expr) Equal(other Command) bool {
	c, ok := other.(Expr)
	if !ok {
		return ok
	}
	return e.eval.Equal(c.eval)
}

const (
	bindLowest int = iota
	bindShift
	bindPlus
	bindMul
	bindPrefix
)

var bindings = map[Kind]int{
	TokAdd:        bindPlus,
	TokSub:        bindPlus,
	TokMul:        bindMul,
	TokDiv:        bindMul,
	TokMod:        bindMul,
	TokLeftShift:  bindShift,
	TokRightShift: bindShift,
}

func bindPower(k Kind) int {
	p, ok := bindings[k]
	if !ok {
		p = bindLowest
	}
	return p
}

type Number struct {
	ident Token
}

func (n Number) Eval(_ Environment) (int, error) {
	return strconv.Atoi(n.ident.Literal)
}

func (n Number) String() string {
	return fmt.Sprintf("number(%s)", n.ident)
}

func (n Number) Equal(other Evaluator) bool {
	c, ok := other.(Number)
	if !ok {
		return ok
	}
	return Compare(n.ident, c.ident)
}

type Identifier struct {
	ident Token
}

func (i Identifier) Eval(env Environment) (int, error) {
	w := env.Resolve(i.ident.Literal)
	return strconv.Atoi(w)
}

func (i Identifier) String() string {
	return fmt.Sprintf("identifier(%s)", i.ident)
}

func (i Identifier) Equal(other Evaluator) bool {
	c, ok := other.(Identifier)
	if !ok {
		return ok
	}
	return Compare(i.ident, c.ident)
}

type Prefix struct {
	op    Kind
	right Evaluator
}

func (p Prefix) Eval(env Environment) (int, error) {
	right, _ := p.right.Eval(env)
	switch p.op {
	case TokSub:
		right = -right
	default:
	}
	return right, nil
}

func (p Prefix) String() string {
	return fmt.Sprintf("prefix(op: %s, right: %s)", p.op, p.right)
}

func (p Prefix) Equal(other Evaluator) bool {
	c, ok := other.(Prefix)
	if !ok {
		return ok
	}
	return p.op == c.op && p.right.Equal(c.right)
}

type Infix struct {
	op    Kind
	left  Evaluator
	right Evaluator
}

func (i Infix) Eval(env Environment) (int, error) {
	left, _ := i.left.Eval(env)
	right, _ := i.right.Eval(env)

	var result int
	switch i.op {
	case TokAdd:
		result = left + right
	case TokSub:
		result = left - right
	case TokMul:
		result = left * right
	case TokDiv:
		result = left / right
	case TokMod:
		result = left % right
	case TokLeftShift:
		result = left >> right
	case TokRightShift:
		result = left << right
	default:
		return 0, fmt.Errorf("infix: unsupported operation %s", i.op)
	}
	return result, nil
}

func (i Infix) String() string {
	return fmt.Sprintf("infix(op: %s, left: %s, right: %s)", i.op, i.left, i.right)
}

func (i Infix) Equal(other Evaluator) bool {
	c, ok := other.(Infix)
	if !ok {
		return ok
	}
	return i.op == c.op && i.left.Equal(i.left) && i.right.Equal(c.right)
}
