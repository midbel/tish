package words

import (
	// "bytes"
	// "context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	// "github.com/midbel/shlex"
	"github.com/midbel/tish/internal/token"
)

var ErrExpansion = errors.New("bad expansion")

type Expander interface {
	Expand(env Environment, top bool) ([]string, error)
	IsQuoted() bool
}

type ExpandSub struct {
	List   []Executer
	Quoted bool
}

func (e ExpandSub) IsQuoted() bool {
	return e.Quoted
}

func (e ExpandSub) Expand(env Environment, top bool) ([]string, error) {
	// sh, ok := env.(*Shell)
	// if !ok {
	// 	return nil, fmt.Errorf("substitution can not expanded")
	// }
	// var (
	// 	err error
	// 	buf bytes.Buffer
	// )
	// sh, _ = sh.Subshell()
	// sh.SetOut(&buf)
	//
	// for i := range e.List {
	// 	if err = sh.execute(context.TODO(), e.List[i]); err != nil {
	// 		return nil, err
	// 	}
	// }
	// return shlex.Split(&buf)
	return nil, fmt.Errorf("to be implemented")
}

type ExpandList struct {
	List   []Expander
	Quoted bool
}

func (e ExpandList) IsQuoted() bool {
	return e.Quoted
}

func (e ExpandList) Expand(env Environment, top bool) ([]string, error) {
	var str []string
	for i := range e.List {
		ws, err := e.List[i].Expand(env, false)
		if err != nil {
			return nil, err
		}
		if top && !e.List[i].IsQuoted() {
			ws = expandList(ws)
		}
		str = append(str, ws...)
	}
	return str, nil
}

func (e *ExpandList) Pop() Expander {
	n := len(e.List)
	if n == 0 {
		return nil
	}
	n--
	x := e.List[n]
	e.List = e.List[:n]
	return x
}

type ExpandMulti struct {
	List   []Expander
	Quoted bool
}

func (m ExpandMulti) IsQuoted() bool {
	return m.Quoted
}

func (m ExpandMulti) Expand(env Environment, top bool) ([]string, error) {
	var words []string
	for _, w := range m.List {
		ws, err := w.Expand(env, false)
		if err != nil {
			return nil, err
		}
		words = append(words, ws...)
	}
	str := strings.Join(words, "")
	if top && !m.IsQuoted() {
		return expandFilename(str), nil
	}
	return []string{str}, nil
}

func (m *ExpandMulti) Pop() Expander {
	n := len(m.List)
	if n == 0 {
		return nil
	}
	n--
	x := m.List[n]
	m.List = m.List[:n]
	return x
}

func (m ExpandMulti) Expander() Expander {
	if len(m.List) == 1 {
		return m.List[0]
	}
	return m
}

type ExpandMath struct {
	List   []Expr
	Quoted bool
}

func (e ExpandMath) Expand(env Environment, _ bool) ([]string, error) {
	var (
		ret float64
		err error
	)
	for i := range e.List {
		ret, err = e.List[i].Eval(env)
		if err != nil {
			return nil, err
		}
	}
	str := strconv.FormatFloat(ret, 'f', -1, 64)
	return []string{str}, nil
}

func (e ExpandMath) IsQuoted() bool {
	return e.Quoted
}

type ExpandWord struct {
	Literal string
	Quoted  bool
}

func CreateWord(str string, quoted bool) ExpandWord {
	return ExpandWord{
		Literal: str,
		Quoted:  quoted,
	}
}

func (w ExpandWord) IsQuoted() bool {
	return w.Quoted
}

func (w ExpandWord) Expand(env Environment, top bool) ([]string, error) {
	if w.Quoted || !top {
		return []string{w.Literal}, nil
	}
	return expandFilename(w.Literal), nil
}

type ExpandRedirect struct {
	Expander
	Quoted bool
	Type   rune
}

func CreateRedirect(e Expander, kind rune) ExpandRedirect {
	return ExpandRedirect{
		Expander: e,
		Type:     kind,
	}
}

func (e ExpandRedirect) IsQuoted() bool {
	return e.Quoted
}

func (e ExpandRedirect) Expand(env Environment, _ bool) ([]string, error) {
	str, err := e.Expander.Expand(env, false)
	if err != nil {
		return nil, err
	}
	if len(str) != 1 {
		return nil, fmt.Errorf("can only redirect to one file")
	}
	return str, nil
}

type ExpandListBrace struct {
	Prefix Expander
	Suffix Expander
	Words  []Expander
}

func (_ ExpandListBrace) IsQuoted() bool {
	return false
}

func (b ExpandListBrace) Expand(env Environment, _ bool) ([]string, error) {
	var (
		prefix []string
		suffix []string
		words  []string
		err    error
	)
	if b.Prefix != nil {
		if prefix, err = b.Prefix.Expand(env, false); err != nil {
			return nil, err
		}
	}
	if b.Suffix != nil {
		if suffix, err = b.Suffix.Expand(env, false); err != nil {
			return nil, err
		}
	}
	for i := range b.Words {
		str, err := b.Words[i].Expand(env, false)
		if err != nil {
			return nil, err
		}
		words = append(words, str...)
	}
	return combineStrings(words, prefix, suffix), nil
}

type ExpandRangeBrace struct {
	Prefix Expander
	Suffix Expander
	Pad    int
	From   int
	To     int
	Step   int
}

func (_ ExpandRangeBrace) IsQuoted() bool {
	return false
}

func (b ExpandRangeBrace) Expand(env Environment, _ bool) ([]string, error) {
	var (
		prefix []string
		suffix []string
		words  []string
		err    error
	)
	if b.Prefix != nil {
		if prefix, err = b.Prefix.Expand(env, false); err != nil {
			return nil, err
		}
	}
	if b.Suffix != nil {
		if suffix, err = b.Suffix.Expand(env, false); err != nil {
			return nil, err
		}
	}
	if b.Step == 0 {
		b.Step = 1
	}
	cmp := func(from, to int) bool {
		return from <= to
	}
	if b.From > b.To {
		cmp = func(from, to int) bool {
			return from >= to
		}
		if b.Step > 0 {
			b.Step = -b.Step
		}
	}
	for cmp(b.From, b.To) {
		str := strconv.Itoa(b.From)
		if z := len(str); b.Pad > 0 && z < b.Pad {
			str = fmt.Sprintf("%s%s", strings.Repeat("0", b.Pad-z), str)
		}
		words = append(words, str)
		b.From += b.Step
	}
	return combineStrings(words, prefix, suffix), nil
}

type ExpandVar struct {
	Ident  string
	Quoted bool
}

func CreateVariable(ident string, quoted bool) ExpandVar {
	return ExpandVar{
		Ident:  ident,
		Quoted: quoted,
	}
}

func (v ExpandVar) IsQuoted() bool {
	return v.Quoted
}

func (v ExpandVar) Expand(env Environment, _ bool) ([]string, error) {
	str, err := env.Resolve(v.Ident)
	if err != nil {
		return nil, err
	}
	if v.Quoted && len(str) > 0 {
		str[0] = strings.Join(str, " ")
		str = str[:1]
	}
	return str, nil
}

func (v ExpandVar) Eval(env Environment) (float64, error) {
	str, err := v.Expand(env, false)
	if err != nil {
		return 0, err
	}
	if len(str) != 1 {
		return 0, fmt.Errorf("expansion returns too many words")
	}
	return strconv.ParseFloat(str[0], 64)
}

type ExpandLength struct {
	Ident string
}

func (_ ExpandLength) IsQuoted() bool {
	return false
}

func (v ExpandLength) Expand(env Environment, _ bool) ([]string, error) {
	var (
		ws, err = env.Resolve(v.Ident)
		sz      int
	)
	if err != nil {
		return nil, err
	}
	for i := range ws {
		sz += len(ws[i])
	}
	s := strconv.Itoa(sz)
	return []string{s}, nil
}

type ExpandReplace struct {
	Ident  string
	From   string
	To     string
	What   rune
	Quoted bool
}

func (v ExpandReplace) IsQuoted() bool {
	return v.Quoted
}

func (v ExpandReplace) Expand(env Environment, _ bool) ([]string, error) {
	str, err := env.Resolve(v.Ident)
	if err != nil {
		return nil, err
	}
	switch v.What {
	case token.Replace:
		str = v.replace(str)
	case token.ReplaceAll:
		str = v.replaceAll(str)
	case token.ReplacePrefix:
		str = v.replacePrefix(str)
	case token.ReplaceSuffix:
		str = v.replaceSuffix(str)
	}
	return str, nil
}

func (v ExpandReplace) replace(str []string) []string {
	for i := range str {
		str[i] = strings.Replace(str[i], v.From, v.To, 1)
	}
	return str
}

func (v ExpandReplace) replaceAll(str []string) []string {
	for i := range str {
		str[i] = strings.ReplaceAll(str[i], v.From, v.To)
	}
	return str
}

func (v ExpandReplace) replacePrefix(str []string) []string {
	return v.replace(str)
}

func (v ExpandReplace) replaceSuffix(str []string) []string {
	return v.replace(str)
}

type ExpandTrim struct {
	Ident  string
	Trim   string
	What   rune
	Quoted bool
}

func (v ExpandTrim) IsQuoted() bool {
	return v.Quoted
}

func (v ExpandTrim) Expand(env Environment, _ bool) ([]string, error) {
	str, err := env.Resolve(v.Ident)
	if err != nil {
		return nil, err
	}
	switch v.What {
	case token.TrimSuffix:
		str = v.trimSuffix(str)
	case token.TrimSuffixLong:
		str = v.trimSuffixLong(str)
	case token.TrimPrefix:
		str = v.trimPrefix(str)
	case token.TrimPrefixLong:
		str = v.trimPrefixLong(str)
	}
	return str, nil
}

func (v ExpandTrim) trimSuffix(str []string) []string {
	for i := range str {
		str[i] = strings.TrimSuffix(str[i], v.Trim)
	}
	return str
}

func (v ExpandTrim) trimSuffixLong(str []string) []string {
	for i := range str {
		for strings.HasSuffix(str[i], v.Trim) {
			str[i] = strings.TrimSuffix(str[i], v.Trim)
		}
	}
	return str
}

func (v ExpandTrim) trimPrefix(str []string) []string {
	for i := range str {
		str[i] = strings.TrimPrefix(str[i], v.Trim)
	}
	return str
}

func (v ExpandTrim) trimPrefixLong(str []string) []string {
	for i := range str {
		for strings.HasPrefix(str[i], v.Trim) {
			str[i] = strings.TrimPrefix(str[i], v.Trim)
		}
	}
	return str
}

type ExpandSlice struct {
	Ident  string
	Offset int
	Size   int
	Quoted bool
}

func (v ExpandSlice) IsQuoted() bool {
	return v.Quoted
}

func (v ExpandSlice) Expand(env Environment, _ bool) ([]string, error) {
	str, err := env.Resolve(v.Ident)
	if err == nil {
		str = v.expandSlice(str)
	}
	return str, err
}

func (v ExpandSlice) expandSlice(str []string) []string {
	var list []string
	for i := range str {
		var (
			siz = len(str[i])
			off = v.Offset
			cut = v.Size
		)
		if siz == 0 {
			list = append(list, str[i])
			continue
		}
		off = normOffset(off, siz)
		off, cut = normCut(off, cut, siz)

		list = append(list, str[i][off:off+cut])
	}
	return list
}

func normCut(off, cut, siz int) (int, int) {
	if cut > 0 {
		if off+cut > siz {
			cut = siz - off
		}
		return off, cut
	}
	if cut < 0 {
		if off == 0 {
			off = siz
		}
		if off+cut < 0 {
			return 0, off
		}
		return off + cut, -cut
	}
	return off, siz - off
}

func normOffset(off, siz int) int {
	if off < 0 {
		off = siz + off
		if off < 0 {
			off = 0
		}
		return off
	}
	if off > siz {
		return siz
	}
	return off
}

type ExpandPad struct {
	Ident  string
	With   string
	Len    int
	What   rune
	Quoted bool
}

func (v ExpandPad) IsQuoted() bool {
	return v.Quoted
}

func (v ExpandPad) Expand(env Environment, _ bool) ([]string, error) {
	str, err := env.Resolve(v.Ident)
	if err != nil || len(str) >= v.Len {
		return str, err
	}
	for i := range str {
		var (
			diff = v.Len - len(str[i])
			fill = strings.Repeat(v.With, diff)
			ori  = str[i]
		)
		if v.What == token.PadRight {
			fill, ori = ori, fill
		}
		str[i] = fmt.Sprintf("%s%s", fill, ori)
	}
	return str, nil
}

var (
	lowerA  byte = 'a'
	lowerZ  byte = 'z'
	upperA  byte = 'A'
	upperZ  byte = 'Z'
	deltaLU byte = 32
)

type ExpandLower struct {
	Ident  string
	All    bool
	Quoted bool
}

func (v ExpandLower) IsQuoted() bool {
	return v.Quoted
}

func (v ExpandLower) Expand(env Environment, _ bool) ([]string, error) {
	str, err := env.Resolve(v.Ident)
	if err != nil {
		return nil, err
	}
	if v.All {
		str = v.lowerAll(str)
	} else {
		str = v.lowerFirst(str)
	}
	return str, nil
}

func (v ExpandLower) lowerFirst(str []string) []string {
	for i := range str {
		if len(str) == 0 {
			continue
		}
		b := []byte(str[i])
		if b[0] >= upperA && b[0] <= upperZ {
			b[0] += deltaLU
		}
		str[i] = string(b)
	}
	return str
}

func (v ExpandLower) lowerAll(str []string) []string {
	for i := range str {
		str[i] = strings.ToLower(str[i])
	}
	return str
}

type ExpandUpper struct {
	Ident  string
	All    bool
	Quoted bool
}

func (v ExpandUpper) IsQuoted() bool {
	return v.Quoted
}

func (v ExpandUpper) Expand(env Environment, _ bool) ([]string, error) {
	str, err := env.Resolve(v.Ident)
	if err != nil {
		return nil, err
	}
	if v.All {
		str = v.upperAll(str)
	} else {
		str = v.upperFirst(str)
	}
	return str, nil
}

func (v ExpandUpper) upperFirst(str []string) []string {
	for i := range str {
		if len(str) == 0 {
			continue
		}
		b := []byte(str[i])
		if b[0] >= lowerA && b[0] <= lowerZ {
			b[0] -= deltaLU
		}
		str[i] = string(b)
	}
	return str
}

func (v ExpandUpper) upperAll(str []string) []string {
	for i := range str {
		str[i] = strings.ToUpper(str[i])
	}
	return str
}

type ExpandValIfUnset struct {
	Ident  string
	Value  string
	Quoted bool
}

func CreateValIfUnset(ident, value string, quoted bool) ExpandValIfUnset {
	return ExpandValIfUnset{
		Ident:  ident,
		Value:  value,
		Quoted: quoted,
	}
}

func (v ExpandValIfUnset) IsQuoted() bool {
	return v.Quoted
}

func (v ExpandValIfUnset) Expand(env Environment, _ bool) ([]string, error) {
	str, err := env.Resolve(v.Ident)
	if err == nil && len(str) > 0 {
		return str, nil
	}
	return []string{v.Value}, nil
}

type ExpandSetValIfUnset struct {
	Ident  string
	Value  string
	Quoted bool
}

func CreateSetValIfUnset(ident, value string, quoted bool) ExpandSetValIfUnset {
	return ExpandSetValIfUnset{
		Ident:  ident,
		Value:  value,
		Quoted: quoted,
	}
}

func (v ExpandSetValIfUnset) IsQuoted() bool {
	return v.Quoted
}

func (v ExpandSetValIfUnset) Expand(env Environment, _ bool) ([]string, error) {
	str, err := env.Resolve(v.Ident)
	if err != nil {
		str = []string{v.Value}
		env.Define(v.Ident, str)
	}
	return str, nil
}

type ExpandValIfSet struct {
	Ident  string
	Value  string
	Quoted bool
}

func CreateExpandValIfSet(ident, value string, quoted bool) ExpandValIfSet {
	return ExpandValIfSet{
		Ident:  ident,
		Value:  value,
		Quoted: quoted,
	}
}

func (v ExpandValIfSet) IsQuoted() bool {
	return v.Quoted
}

func (v ExpandValIfSet) Expand(env Environment, _ bool) ([]string, error) {
	str, err := env.Resolve(v.Ident)
	if err == nil {
		str = []string{v.Value}
	}
	return str, nil
}

type ExpandExitIfUnset struct {
	Ident  string
	Value  string
	Quoted bool
}

func CreateExpandExitIfUnset(ident, value string, quoted bool) ExpandExitIfUnset {
	return ExpandExitIfUnset{
		Ident:  ident,
		Value:  value,
		Quoted: quoted,
	}
}

func (v ExpandExitIfUnset) IsQuoted() bool {
	return v.Quoted
}

func (v ExpandExitIfUnset) Expand(env Environment, _ bool) ([]string, error) {
	return nil, nil
}

func combineStrings(words, prefix, suffix []string) []string {
	if len(prefix) == 0 && len(suffix) == 0 {
		return words
	}
	var (
		tmp strings.Builder
		str = combineStringsWith(&tmp, words, prefix)
	)
	return combineStringsWith(&tmp, suffix, str)
}

func combineStringsWith(ws *strings.Builder, all, with []string) []string {
	if len(with) == 0 {
		return all
	}
	if len(all) == 0 {
		return with
	}
	var str []string
	for i := range with {
		for j := range all {
			ws.WriteString(with[i])
			ws.WriteString(all[j])
			str = append(str, ws.String())
			ws.Reset()
		}
	}
	return str
}

func expandList(str []string) []string {
	var list []string
	for i := range str {
		list = append(list, expandFilename(str[i])...)
	}
	return list
}

func expandFilename(str string) []string {
	if strings.HasPrefix(str, "~") {
		return expandTilde(str)
	}
	if strings.ContainsAny(str, "[?*") {
		dir, file := filepath.Split(str)
		str = filepath.Join(filepath.Dir(dir), file)
		return expandPath(str)
	}
	return []string{str}
}

func expandTilde(str string) []string {
	// TODO: replace tilde by home directory of current user
	return []string{str}
}

func expandPath(str string) []string {
	list, err := filepath.Glob(str)
	if err != nil || len(list) == 0 {
		list = append(list[:0], str)
	}
	return list
}
