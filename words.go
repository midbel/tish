package tish

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type Word interface {
	Command
	Expand(Environment) []string
}

func CompareWords(fst, snd Word) bool {
	if fst == nil && snd == nil {
		return true
	}
	return fst != nil && snd != nil && fst.Equal(snd)
}

type Evaluator interface {
	fmt.Stringer
	Eval(Environment) (int, error)
	Equal(Evaluator) bool
}

type WordList struct {
	words []Word
}

func createList(ws ...Word) Word {
	return WordList{ws}
}

func (w WordList) Expand(env Environment) []string {
	str := w.expand(env)
	return []string{str}
}

func (w WordList) expand(env Environment) string {
	ws := make([]string, 0, len(w.words))
	for _, w := range w.words {
		str := w.Expand(env)
		ws = append(ws, str...)
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
		if !CompareWords(w.words[i], c.words[i]) {
			return false
		}
	}
	return true
}

func (w WordList) asWord() Word {
	switch len(w.words) {
	case 0:
		return nil
	case 1:
		return w.words[0]
	default:
		return w
	}
}

type Literal struct {
	token Token
}

func createLiteral(tok Token) Word {
	return Literal{tok}
}

func (i Literal) Expand(env Environment) []string {
	str := i.expand(env)
	return []string{str}
}

func (i Literal) expand(env Environment) string {
	var str string
	switch i.token.Type {
	case TokLiteral, TokNumber:
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

func (s Slice) Expand(env Environment) []string {
	str := s.expand(env)
	return []string{str}
}

func (s Slice) expand(env Environment) string {
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

func (t Trim) Expand(env Environment) []string {
	str := t.expand(env)
	return []string{str}
}

func (t Trim) expand(env Environment) string {
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

func (r Replace) Expand(env Environment) []string {
	str := r.expand(env)
	return []string{str}
}

func (r Replace) expand(env Environment) string {
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

func (t Transform) Expand(env Environment) []string {
	str := t.expand(env)
	return []string{str}
}

func (t Transform) expand(env Environment) string {
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

func (e Length) Expand(env Environment) []string {
	str := e.expand(env)
	return []string{str}
}

func (e Length) expand(env Environment) string {
	str := env.Resolve(e.ident.Literal)
	return strconv.Itoa(len(str))
}

type Serie struct {
	prefix Word
	suffix Word
	words  []Word
}

func (s Serie) Expand(env Environment) []string {
	return s.expand(env)
}

func (s Serie) expand(env Environment) []string {
	ws := make([]string, 0, len(s.words))
	for i := range s.words {
		str := s.words[i].Expand(env)
		ws = append(ws, str...)
	}
	ws = s.expandPrefix(ws, env)
	return s.expandSuffix(ws, env)
}

func (s Serie) expandPrefix(ws []string, env Environment) []string {
	if s.prefix == nil {
		return ws
	}
	var (
		ps = s.prefix.Expand(env)
		vs = make([]string, 0, len(ps)*len(ws))
	)
	for i := range ws {
		for j := range ps {
			vs = append(vs, ps[j]+ws[i])
		}
	}
	return vs
}

func (s Serie) expandSuffix(ws []string, env Environment) []string {
	if s.suffix == nil {
		return ws
	}
	var (
		ps = s.suffix.Expand(env)
		vs = make([]string, 0, len(ps)*len(ws))
	)
	for i := range ws {
		for j := range ps {
			vs = append(vs, ws[i]+ps[j])
		}
	}
	return vs
}

func (s Serie) String() string {
	ws := make([]string, len(s.words))
	for i := range s.words {
		ws[i] = s.words[i].String()
	}
	return fmt.Sprintf("serie(words: %s, prefix: %v, suffix: %v)", strings.Join(ws, ", "), s.prefix, s.suffix)
}

func (s Serie) Equal(other Command) bool {
	c, ok := other.(Serie)
	if !ok {
		return ok
	}
	if len(s.words) != len(c.words) {
		return false
	}
	for i := range s.words {
		if !CompareWords(s.words[i], c.words[i]) {
			return false
		}
	}
	return CompareWords(s.prefix, c.prefix) && CompareWords(s.suffix, c.suffix)
}

type Range struct {
	prefix Word
	suffix Word
	first  Word
	last   Word
	incr   Word
}

func (r Range) Expand(env Environment) []string {
	return r.expand(env)
}

func (r Range) expand(env Environment) []string {
	first, last, incr := r.expandLimits(env)
	if incr == 0 {
		return nil
	}
	var isLess func(int, int) bool
	if last < first {
		incr = -incr
		isLess = func(fst, lst int) bool { return fst >= lst }
	} else {
		isLess = func(fst, lst int) bool { return fst <= lst }
	}
	var (
		vs  = make([]string, 0, 10)
		pad = r.computePadding(env)
	)
	for isLess(first, last) {
		str := strconv.Itoa(first)
		if n := len(str); pad > 0 && n < pad {
			str = strings.Repeat("0", pad-n) + str
		}
		vs = append(vs, str)
		first += incr
	}
	vs = r.expandPrefix(vs, env)
	return r.expandSuffix(vs, env)
}

func (r Range) expandPrefix(ws []string, env Environment) []string {
	if r.prefix == nil {
		return ws
	}
	var (
		ps = r.prefix.Expand(env)
		vs = make([]string, 0, len(ps)*len(ws))
	)
	for i := range ws {
		for j := range ps {
			vs = append(vs, ps[j]+ws[i])
		}
	}
	return vs
}

func (r Range) expandSuffix(ws []string, env Environment) []string {
	if r.suffix == nil {
		return ws
	}
	var (
		ps = r.suffix.Expand(env)
		vs = make([]string, 0, len(ps)*len(ws))
	)
	for i := range ws {
		for j := range ps {
			vs = append(vs, ws[i]+ps[j])
		}
	}
	return vs
}

func (r Range) expandLimits(env Environment) (int, int, int) {
	convert := func(w Word) int {
		is := w.Expand(env)
		if len(is) != 1 {
			return 0
		}
		v, _ := strconv.Atoi(is[0])
		return v
	}
	return convert(r.first), convert(r.last), convert(r.incr)
}

func (r Range) computePadding(env Environment) int {
	es := r.first.Expand(env)
	if len(es) != 1 {
		return 0
	}
	str := es[0]
	if len(str) <= 1 {
		return 0
	}
	var pad int
	for pad < len(str)-1 && str[pad] == '0' {
		pad++
	}
	return pad + 1
}

func (r Range) String() string {
	return fmt.Sprintf("range(first: %s, last: %s, incr: %s, prefix: %v, suffix: %v)", r.first, r.last, r.incr, r.prefix, r.suffix)
}

func (r Range) Equal(other Command) bool {
	c, ok := other.(Range)
	if !ok {
		return ok
	}
	ok = CompareWords(r.first, c.first) && CompareWords(r.last, c.last) && CompareWords(r.incr, c.incr)
	if !ok {
		return ok
	}
	return CompareWords(r.prefix, c.prefix) && CompareWords(r.suffix, c.suffix)
}

type Expr struct {
	eval Evaluator
}

func createExpr(es ...Evaluator) Word {
	if len(es) == 1 {
		return Expr{es[0]}
	}
	return Expr{EvalList{es}}
}

func (e Expr) Expand(env Environment) []string {
	str := e.expand(env)
	return []string{str}
}

func (e Expr) expand(env Environment) string {
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
	bindAssign
	bindOr
	bindAnd
	bindBinOr
	bindBinXor
	bindBinAnd
	bindEq
	bindCmp
	bindShift
	bindPlus
	bindMul
	bindExp
	bindIncr
	bindPrefix
)

var bindings = map[Kind]int{
	TokIncr:             bindIncr,
	TokDecr:             bindIncr,
	TokAdd:              bindPlus,
	TokSub:              bindPlus,
	TokMul:              bindMul,
	TokDiv:              bindMul,
	TokMod:              bindMul,
	TokExponent:         bindExp,
	TokBinAnd:           bindBinAnd,
	TokBinOr:            bindBinOr,
	TokBinXor:           bindBinXor,
	TokLeftShift:        bindShift,
	TokRightShift:       bindShift,
	TokLesser:           bindCmp,
	TokLessEq:           bindCmp,
	TokGreater:          bindCmp,
	TokGreatEq:          bindCmp,
	TokAnd:              bindAnd,
	TokOr:               bindOr,
	TokEqual:            bindEq,
	TokNotEqual:         bindEq,
	TokAssign:           bindAssign,
	TokAddAssign:        bindAssign,
	TokSubAssign:        bindAssign,
	TokMulAssign:        bindAssign,
	TokDivAssign:        bindAssign,
	TokModAssign:        bindAssign,
	TokLeftShiftAssign:  bindAssign,
	TokRightShiftAssign: bindAssign,
	TokBinAndAssign:     bindAssign,
	TokBinOrAssign:      bindAssign,
	TokBinXorAssign:     bindAssign,
}

func bindPower(k Kind) int {
	p, ok := bindings[k]
	if !ok {
		p = bindLowest
	}
	return p
}

type EvalList struct {
	evals []Evaluator
}

func (e EvalList) Eval(env Environment) (int, error) {
	var (
		res int
		err error
	)
	for _, e := range e.evals {
		res, err = e.Eval(env)
		if err == nil {
			break
		}
	}
	return res, err
}

func (e EvalList) String() string {
	es := make([]string, len(e.evals))
	for i := range e.evals {
		es[i] = e.evals[i].String()
	}
	return fmt.Sprintf("evallist(%s)", strings.Join(es, ", "))
}

func (e EvalList) Equal(other Evaluator) bool {
	c, ok := other.(EvalList)
	if !ok {
		return ok
	}
	if len(e.evals) != len(c.evals) {
		return false
	}
	for i := range e.evals {
		if !e.evals[i].Equal(c.evals[i]) {
			return false
		}
	}
	return true
}

func (e EvalList) asEvaluator() Evaluator {
	if len(e.evals) == 1 {
		return e.evals[0]
	}
	return e
}

type Number struct {
	ident Token
}

func createNumber(tok Token) Evaluator {
	return Number{tok}
}

func (n Number) Eval(_ Environment) (int, error) {
	x, err := strconv.ParseInt(n.ident.Literal, 0, 64)
	return int(x), err
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

func createIdentifier(tok Token) Evaluator {
	return Identifier{tok}
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

func createPrefix(right Evaluator, op Kind) Evaluator {
	return Prefix{op: op, right: right}
}

func (p Prefix) Eval(env Environment) (int, error) {
	right, _ := p.right.Eval(env)
	var err error
	switch p.op {
	case TokNot:
		if right == 0 {
			right = 1
		} else {
			right = 0
		}
	case TokIncr:
		v, ok := p.right.(Identifier)
		if !ok {
			err = fmt.Errorf("expected identifier on right side of increment operator")
			break
		}
		right++
		err = env.Define(v.ident.Literal, strconv.Itoa(right))
	case TokDecr:
		v, ok := p.right.(Identifier)
		if !ok {
			err = fmt.Errorf("expected identifier on right side of decrement operator")
			break
		}
		right--
		err = env.Define(v.ident.Literal, strconv.Itoa(right))
	case TokAdd:
	case TokSub:
		right = -right
	case TokBinNot:
		right = ^right
	default:
	}
	return right, err
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

func createInfix(left, right Evaluator, op Kind) Evaluator {
	return Infix{op: op, left: left, right: right}
}

func (i Infix) Eval(env Environment) (int, error) {
	left, _ := i.left.Eval(env)
	right, _ := i.right.Eval(env)

	var (
		res int
		err error
	)
	switch i.op {
	case TokAssign:
		res = right
	case TokAdd, TokAddAssign:
		res = left + right
	case TokSub, TokSubAssign:
		res = left - right
	case TokMul, TokMulAssign:
		res = left * right
	case TokDiv, TokDivAssign:
		if right == 0 {
			err = fmt.Errorf("division by zero")
			break
		}
		res = left / right
	case TokMod, TokModAssign:
		res = left % right
	case TokExponent:
		pow := math.Pow(float64(left), float64(right))
		res = int(pow)
	case TokLeftShift, TokLeftShiftAssign:
		res = left >> right
	case TokRightShift, TokRightShiftAssign:
		res = left << right
	case TokBinAnd, TokBinAndAssign:
		res = left & right
	case TokBinOr, TokBinOrAssign:
		res = left | right
	case TokBinXor, TokBinXorAssign:
		res = left ^ right
	case TokEqual:
		if left != right {
			res++
		}
	case TokNotEqual:
		if left == right {
			res++
		}
	case TokLesser:
		if left >= right {
			res++
		}
	case TokLessEq:
		if left > right {
			res++
		}
	case TokGreater:
		if left <= right {
			res++
		}
	case TokGreatEq:
		if left < right {
			res++
		}
	case TokAnd:
		if left != 0 {
			res = right
		}
	case TokOr:
		if left == 0 {
			res = right
		}
	default:
		err = fmt.Errorf("infix: unsupported operation %s", i.op)
	}
	if err == nil && i.op.IsAssign() {
		res, err = i.assign(env, res)
	}
	return res, err
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

func (i Infix) assign(env Environment, result int) (int, error) {
	v, ok := i.left.(Identifier)
	if !ok {
		return 1, fmt.Errorf("expected identifier on left side of assignment")
	}
	return 0, env.Define(v.ident.Literal, strconv.Itoa(result))
}
