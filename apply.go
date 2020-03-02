package tish

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Apply interface {
	Apply(string, Environment) ([]string, error)
	Equal(Apply) bool
	fmt.Stringer
}

type substring struct {
	offset Word
	length Word
}

func Substring(offset, length Word) Apply {
	return substring{
		offset: offset,
		length: length,
	}
}

func (s substring) Apply(ident string, e Environment) ([]string, error) {
	vs, err := e.Resolve(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}

	offset, length, err := s.expandValues(e)
	if err != nil {
		return nil, err
	}
	if offset == 0 && length == 0 {
		return []string{}, nil
	}

	var str = vs[0]
	if offset < 0 {
		offset = len(str) + offset
	}
	if offset < 0 || offset >= len(str) {
		return vs, nil
	}

	switch str = str[offset:]; {
	case length == 0:
	case length < 0:
		if sz := len(str) + length; sz >= 0 {
			str = str[:sz]
		}
	case length > 0:
		if length < len(str) {
			str = str[:length]
		}
	}
	return []string{str}, nil
}

func (s substring) Equal(a Apply) bool {
	other, ok := a.(substring)
	if !ok {
		return ok
	}
	return s.offset.Equal(other.offset) && s.length.Equal(other.length)
}

func (s substring) String() string {
	return fmt.Sprintf("substring(%d, %d)", s.offset, s.length)
}

func (s substring) expandValues(e Environment) (int, int, error) {
	var offset, length int

	if off, ok := s.offset.(Number); !ok {
		vs, err := s.offset.Expand(e)
		if err != nil {
			return offset, length, err
		}
		if len(vs) > 0 {
			offset, err = strconv.Atoi(vs[0])
			if err != nil {
				return offset, length, err
			}
		}
	} else {
		offset = int(off)
	}
	if lgt, ok := s.length.(Number); !ok {
		vs, err := s.length.Expand(e)
		if err != nil {
			return offset, length, err
		}
		if len(vs) > 0 {
			length, err = strconv.Atoi(vs[0])
			if err != nil {
				return offset, length, err
			}
		}
	} else {
		length = int(lgt)
	}
	return offset, length, nil
}

type length struct{}

func Length() Apply {
	return length{}
}

func (n length) Apply(ident string, e Environment) ([]string, error) {
	vs, err := e.Resolve(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}
	x := len(vs[0])
	return []string{strconv.Itoa(x)}, nil
}

func (n length) Equal(a Apply) bool {
	_, ok := a.(length)
	return ok
}

func (n length) String() string {
	return "length"
}

type trimPrefix struct {
	pattern Word
	longest bool
}

func TrimPrefix(w Word, longest bool) Apply {
	return trimPrefix{pattern: w, longest: longest}
}

func (p trimPrefix) Apply(ident string, e Environment) ([]string, error) {
	vs, err := e.Resolve(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}
	xs, err := p.pattern.Expand(e)
	if err != nil || len(xs) == 0 {
		return nil, err
	}
	str := strings.TrimPrefix(vs[0], xs[0])
	for p.longest && strings.HasPrefix(str, xs[0]) {
		str = strings.TrimPrefix(str, xs[0])
	}
	return []string{str}, nil
}

func (p trimPrefix) Equal(a Apply) bool {
	other, ok := a.(trimPrefix)
	if !ok {
		return ok
	}
	return p.pattern.Equal(other.pattern) && p.longest == other.longest
}

func (p trimPrefix) String() string {
	return fmt.Sprintf("prefix(%s)", p.pattern)
}

type trimSuffix struct {
	pattern Word
	longest bool
}

func TrimSuffix(w Word, longest bool) Apply {
	return trimSuffix{pattern: w, longest: longest}
}

func (s trimSuffix) Apply(ident string, e Environment) ([]string, error) {
	vs, err := e.Resolve(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}
	xs, err := s.pattern.Expand(e)
	if err != nil || len(xs) == 0 {
		return nil, err
	}
	str := strings.TrimSuffix(vs[0], xs[0])
	for s.longest && strings.HasSuffix(str, xs[0]) {
		str = strings.TrimSuffix(str, xs[0])
	}
	return []string{str}, nil
}

func (s trimSuffix) Equal(a Apply) bool {
	other, ok := a.(trimSuffix)
	if !ok {
		return ok
	}
	return s.pattern.Equal(other.pattern) && s.longest == other.longest
}

func (s trimSuffix) String() string {
	return fmt.Sprintf("suffix(%s)", s.pattern)
}

type replace struct {
	src    Word
	dst    Word
	prefix bool
	suffix bool
}

func Replace(src, dst Word) Apply {
	return replace{src: src, dst: dst}
}

func ReplaceAll(src, dst Word) Apply {
	return replace{
		src:    src,
		dst:    dst,
		prefix: true,
		suffix: true,
	}
}

func ReplacePrefix(src, dst Word) Apply {
	return replace{
		src:    src,
		dst:    dst,
		prefix: true,
	}
}

func ReplaceSuffix(src, dst Word) Apply {
	return replace{
		src:    src,
		dst:    dst,
		suffix: true,
	}
}

func (r replace) Apply(ident string, e Environment) ([]string, error) {
	vs, err := e.Resolve(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}
	str := vs[0]
	src, dst, err := r.expandValues(e)
	if err != nil {
		return nil, err
	}
	switch {
	default:
		str = strings.Replace(str, src, dst, 1)
	case r.prefix && r.suffix:
		str = strings.ReplaceAll(str, src, dst)
	case r.prefix && !r.suffix:
		if strings.HasPrefix(str, src) {
			str = dst + strings.TrimPrefix(str, src)
		}
	case r.suffix && !r.prefix:
		if strings.HasSuffix(str, src) {
			str = strings.TrimSuffix(str, src) + dst
		}
	}
	return []string{str}, nil
}

func (r replace) Equal(a Apply) bool {
	other, ok := a.(replace)
	if !ok {
		return ok
	}
	ok = r.prefix == other.prefix && r.suffix == other.suffix
	if !ok {
		return ok
	}
	return r.src.Equal(other.src) && r.dst.Equal(other.dst)
}

func (r replace) String() string {
	return fmt.Sprintf("replace(%s, %s)", r.src, r.dst)
}

func (r replace) expandValues(e Environment) (string, string, error) {
	var src, dst string
	vs, err := r.src.Expand(e)
	if err == nil && len(vs) > 0 {
		src = vs[0]
	}
	vs, err = r.dst.Expand(e)
	if err == nil && len(vs) > 0 {
		dst = vs[0]
	}
	return src, dst, err
}

type lower struct {
	all bool
}

func Lower(all bool) Apply {
	return lower{all}
}

func (o lower) Apply(ident string, e Environment) ([]string, error) {
	vs, err := e.Resolve(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}
	str := vs[0]
	if o.all {
		str = strings.ToLower(str)
	} else {
		rs := []rune(str)
		if unicode.IsUpper(rs[0]) {
			rs[0] = unicode.SimpleFold(rs[0])
		}
		str = string(rs)
	}
	return []string{str}, nil
}

func (o lower) Equal(a Apply) bool {
	other, ok := a.(lower)
	return ok && o.all == other.all
}

func (o lower) String() string {
	return "lower"
}

type upper struct {
	all bool
}

func Upper(all bool) Apply {
	return upper{all}
}

func (u upper) Apply(ident string, e Environment) ([]string, error) {
	vs, err := e.Resolve(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}
	str := vs[0]
	if u.all {
		str = strings.ToUpper(str)
	} else {
		rs := []rune(str)
		if unicode.IsLower(rs[0]) {
			rs[0] = unicode.SimpleFold(rs[0])
		}
		str = string(rs)
	}
	return []string{str}, nil
}

func (u upper) Equal(a Apply) bool {
	other, ok := a.(upper)
	return ok && u.all == other.all
}

func (u upper) String() string {
	return "upper"
}

// ${FOO:=BAR}
type setifundef struct {
	str Word
}

func SetIfUndef(str Word) Apply {
	return setifundef{str}
}

func (s setifundef) Apply(ident string, e Environment) ([]string, error) {
	vs, err := e.Resolve(ident)
	if err != nil {
		vs, err = s.str.Expand(e)
		e.Define(ident, vs)
	}
	return vs, err
}

func (s setifundef) Equal(a Apply) bool {
	other, ok := a.(setifundef)
	return ok && s.str.Equal(other.str)
}

func (s setifundef) String() string {
	return fmt.Sprintf("setifundef(%s)", s.str)
}

// ${FOO:-BAR}
type getifundef struct {
	str Word
}

func GetIfUndef(str Word) Apply {
	return getifundef{str}
}

func (g getifundef) Apply(ident string, e Environment) ([]string, error) {
	vs, err := e.Resolve(ident)
	if err != nil {
		vs, err = g.str.Expand(e)
	}
	return vs, err
}

func (g getifundef) Equal(a Apply) bool {
	other, ok := a.(getifundef)
	return ok && g.str.Equal(other.str)
}

func (g getifundef) String() string {
	return fmt.Sprintf("getifundef(%s)", g.str)
}

// ${FOO:+BAR}
type getifdef struct {
	str Word
}

func GetIfDef(str Word) Apply {
	return getifdef{str}
}

func (g getifdef) Apply(ident string, e Environment) ([]string, error) {
	vs, err := e.Resolve(ident)
	if err == nil {
		vs, err = g.str.Expand(e)
	}
	return vs, err
}

func (g getifdef) Equal(a Apply) bool {
	other, ok := a.(getifdef)
	return ok && g.str.Equal(other.str)
}

func (g getifdef) String() string {
	return fmt.Sprintf("getifdef(%s)", g.str)
}

type identity struct{}

func Identity() Apply {
	return identity{}
}

func (i identity) Apply(ident string, e Environment) ([]string, error) {
	return e.Resolve(ident)
}

func (_ identity) Equal(a Apply) bool {
	_, ok := a.(identity)
	return ok
}

func (i identity) String() string {
	return "identity"
}
