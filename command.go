package tish

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Command interface {
	fmt.Stringer
	Equal(Command) bool
}

type Word interface {
	Command
	Expand(Environment) string
}

type Literal struct {
	tokens []Token
}

func (i Literal) Expand(env Environment) string {
	ws := make([]string, 0, len(i.tokens))
	for _, tok := range i.tokens {
		var str string
		switch tok.Type {
		case TokLiteral:
			str = tok.Literal
		case TokVariable:
			str = env.Resolve(tok.Literal)
		default:
			continue
		}
		ws = append(ws, str)
	}
	return strings.Join(ws, "")
}

func (i Literal) String() string {
	ws := make([]string, len(i.tokens))
	for j := range i.tokens {
		ws[j] = i.tokens[j].Literal
	}
	return fmt.Sprintf("literal(%s)", strings.Join(ws, ""))
}

func (i Literal) Equal(other Command) bool {
	c, ok := other.(Literal)
	if !ok {
		return ok
	}
	if len(i.tokens) != len(c.tokens) {
		return false
	}
	for j := range i.tokens {
		if !Compare(i.tokens[j], c.tokens[j]) {
			return false
		}
	}
	return true
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

func (s Simple) Equal(other Command) bool {
	c, ok := other.(Simple)
	if !ok {
		return ok
	}
	if len(s.env) != len(c.env) {
		return false
	}
	for i := range s.env {
		if !s.env[i].Equal(c.env[i]) {
			return false
		}
	}
	if len(s.words) != len(c.words) {
		return false
	}
	for i := range s.words {
		if !s.words[i].Equal(c.words[i]) {
			return false
		}
	}
	return true
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

func (i List) Equal(other Command) bool {
	c, ok := other.(List)
	if !ok {
		return ok
	}
	if len(i.cmds) != len(c.cmds) {
		return false
	}
	for j := range i.cmds {
		if !i.cmds[j].Equal(c.cmds[j]) {
			return false
		}
	}
	return true
}

type And struct {
	left  Command
	right Command
}

func (a And) String() string {
	return fmt.Sprintf("and(%s, %s)", a.left, a.right)
}

func (a And) Equal(other Command) bool {
	c, ok := other.(And)
	if !ok {
		return ok
	}
	return a.left.Equal(c.left) && a.right.Equal(c.right)
}

type Or struct {
	left  Command
	right Command
}

func (o Or) String() string {
	return fmt.Sprintf("or(%s, %s)", o.left, o.right)
}

func (o Or) Equal(other Command) bool {
	c, ok := other.(Or)
	if !ok {
		return ok
	}
	return o.left.Equal(c.left) && o.right.Equal(c.right)
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

func (c Case) Equal(other Command) bool {
	x, ok := other.(Case)
	if !ok {
		return ok
	}
	if !c.word.Equal(x.word) {
		return false
	}
	if len(c.clauses) != len(x.clauses) {
		return false
	}
	for i := range c.clauses {
		if !c.clauses[i].Equal(x.clauses[i]) {
			return false
		}
	}
	return true
}

type Clause struct {
	pattern []Word
	body    Command
	op      Token
}

func (c Clause) Match(str string, env Environment) bool {
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

func (c Clause) Equal(other Command) bool {
	x, ok := other.(Clause)
	if !ok {
		return ok
	}
	ok = Compare(c.op, x.op)
	if c.body != nil && x.body != nil {
		return c.body.Equal(x.body) && ok
	}
	return (c.body == nil && x.body == nil) && ok
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

func (i If) Equal(other Command) bool {
	c, ok := other.(If)
	if !ok {
		return ok
	}
	if !i.cmd.Equal(c.cmd) {
		return false
	}
	if !i.csq.Equal(c.csq) {
		return false
	}
	if i.alt != nil && c.alt != nil {
		return i.alt.Equal(c.alt)
	}
	return i.alt == nil && c.alt == nil
}

type Until struct {
	cmd  Command
	body Command
}

func (u Until) String() string {
	return fmt.Sprintf("until(cmd: %s, body: %s)", u.cmd, u.body)
}

func (u Until) Equal(other Command) bool {
	c, ok := other.(Until)
	if !ok {
		return ok
	}
	if !u.cmd.Equal(c.cmd) {
		return false
	}
	if u.body != nil && c.body != nil {
		return u.body.Equal(c.body)
	}
	return u.body == nil && c.body == nil
}

type While struct {
	cmd  Command
	body Command
}

func (w While) String() string {
	return fmt.Sprintf("while(cmd: %s, body: %s)", w.cmd, w.body)
}

func (w While) Equal(other Command) bool {
	c, ok := other.(While)
	if !ok {
		return ok
	}
	if !w.cmd.Equal(c.cmd) {
		return false
	}
	if w.body != nil && c.body != nil {
		return w.body.Equal(c.body)
	}
	return w.body == nil && c.body == nil
}

type For struct {
	ident Token
	words []Word
	body  Command
}

func (f For) String() string {
	return fmt.Sprintf("for(words: %s, body: %s)", "", f.body.String())
}

func (f For) Equal(other Command) bool {
	c, ok := other.(For)
	if !ok {
		return ok
	}
	if !Compare(f.ident, c.ident) {
		return false
	}
	if len(f.words) != len(c.words) {
		return false
	}
	for i := range f.words {
		if !f.words[i].Equal(c.words[i]) {
			return false
		}
	}
	if f.body != nil && c.body != nil {
		return f.body.Equal(c.body)
	}
	return f.body == nil && c.body == nil
}

type Break struct{}

func (_ Break) String() string {
	return "break()"
}

func (_ Break) Equal(other Command) bool {
	_, ok := other.(Break)
	return ok
}

type Continue struct{}

func (_ Continue) String() string {
	return "continue()"
}

func (_ Continue) Equal(other Command) bool {
	_, ok := other.(Continue)
	return ok
}

type Assign struct {
	ident Token
	word  Word
}

func (a Assign) String() string {
	return fmt.Sprintf("assign(ident: %s, word: %s)", a.ident, a.word)
}

func (a Assign) Equal(other Command) bool {
	c, ok := other.(Assign)
	if !ok {
		return ok
	}
	return Compare(a.ident, c.ident) && a.word.Equal(c.word)
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
