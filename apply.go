package tish

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Apply interface {
	Apply(string, *Env) ([]string, error)
	fmt.Stringer
}

type substring struct {
	offset int
	length int
}

func Substring(offset, length int) Apply {
	return substring{offset, length}
}

func (s substring) Apply(ident string, e *Env) ([]string, error) {
	vs, err := e.Get(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}

	var (
		offset = s.offset
		str    = vs[0]
	)
	if offset < 0 {
		offset = len(str) + offset
	}
	if offset < 0 || offset >= len(str) {
		return vs, nil
	}
	str = str[offset:]
	switch sz := s.length; {
	case sz == 0:
	case sz < 0:
		if sz = len(str) + sz; sz >= 0 {
			str = str[:sz]
		}
	case sz > 0:
		if sz < len(str) {
			str = str[:sz]
		}
	}
	return []string{str}, nil
}

func (s substring) String() string {
	return fmt.Sprintf("substring(%d, %d)", s.offset, s.length)
}

type length struct{}

func Length() Apply {
	return length{}
}

func (n length) String() string {
	return "length"
}

func (n length) Apply(ident string, e *Env) ([]string, error) {
	vs, err := e.Get(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}
	x := len(vs[0])
	return []string{strconv.Itoa(x)}, nil
}

type trimPrefix struct {
	pattern string
	longest bool
}

func TrimPrefix(str string, longest bool) Apply {
	return trimPrefix{pattern: str, longest: longest}
}

func (p trimPrefix) Apply(ident string, e *Env) ([]string, error) {
	vs, err := e.Get(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}
	str := strings.TrimPrefix(vs[0], p.pattern)
	for p.longest && strings.HasPrefix(str, p.pattern) {
		str = strings.TrimPrefix(str, p.pattern)
	}
	return []string{str}, nil
}

func (p trimPrefix) String() string {
	return fmt.Sprintf("prefix(%s)", p.pattern)
}

type trimSuffix struct {
	pattern string
	longest bool
}

func TrimSuffix(str string, longest bool) Apply {
	return trimSuffix{pattern: str, longest: longest}
}

func (s trimSuffix) Apply(ident string, e *Env) ([]string, error) {
	vs, err := e.Get(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}
	str := strings.TrimSuffix(vs[0], s.pattern)
	for s.longest && strings.HasSuffix(str, s.pattern) {
		str = strings.TrimSuffix(str, s.pattern)
	}
	return []string{str}, nil
}

func (s trimSuffix) String() string {
	return fmt.Sprintf("suffix(%s)", s.pattern)
}

type replace struct {
	src    string
	dst    string
	prefix bool
	suffix bool
}

func Replace(src, dst string) Apply {
	return replace{src: src, dst: dst}
}

func ReplaceAll(src, dst string) Apply {
	return replace{
		src:    src,
		dst:    dst,
		prefix: true,
		suffix: true,
	}
}

func ReplacePrefix(src, dst string) Apply {
	return replace{
		src:    src,
		dst:    dst,
		prefix: true,
	}
}

func ReplaceSuffix(src, dst string) Apply {
	return replace{
		src:    src,
		dst:    dst,
		suffix: true,
	}
}

func (r replace) Apply(ident string, e *Env) ([]string, error) {
	vs, err := e.Get(ident)
	if err != nil || len(vs) == 0 {
		return vs, err
	}
	str := vs[0]
	switch {
	default:
		str = strings.Replace(str, r.src, r.dst, 1)
	case r.prefix && r.suffix:
		str = strings.ReplaceAll(str, r.src, r.dst)
	case r.prefix && !r.suffix:
		if strings.HasPrefix(str, r.src) {
			str = r.dst + strings.TrimPrefix(str, r.src)
		}
	case r.suffix && !r.prefix:
		if strings.HasSuffix(str, r.src) {
			str = strings.TrimSuffix(str, r.src) + r.dst
		}
	}
	return []string{str}, nil
}

func (r replace) String() string {
	return fmt.Sprintf("replace(%s, %s)", r.src, r.dst)
}

type lower struct {
	all bool
}

func Lower(all bool) Apply {
	return lower{all}
}

func (o lower) Apply(ident string, e *Env) ([]string, error) {
	vs, err := e.Get(ident)
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

func (o lower) String() string {
	return "lower"
}

type upper struct {
	all bool
}

func Upper(all bool) Apply {
	return upper{all}
}

func (u upper) Apply(ident string, e *Env) ([]string, error) {
	vs, err := e.Get(ident)
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

func (u upper) String() string {
	return "upper"
}

// ${FOO:=BAR}
type setifundef struct {
	str string
}

func SetIfUndef(str string) Apply {
	return setifundef{str}
}

func (s setifundef) Apply(ident string, e *Env) ([]string, error) {
	vs, err := e.Get(ident)
	if err != nil {
		vs = []string{s.str}
		e.Set(ident, vs)
	}
	return vs, nil
}

func (s setifundef) String() string {
	return fmt.Sprintf("setifundef(%s)", s.str)
}

// ${FOO:-BAR}
type getifundef struct {
	str string
}

func GetIfUndef(str string) Apply {
	return getifundef{str}
}

func (g getifundef) Apply(ident string, e *Env) ([]string, error) {
	vs, err := e.Get(ident)
	if err != nil {
		vs = []string{g.str}
	}
	return vs, nil
}

func (g getifundef) String() string {
	return fmt.Sprintf("getifundef(%s)", g.str)
}

// ${FOO:+BAR}
type getifdef struct {
	str string
}

func GetIfDef(str string) Apply {
	return getifdef{str}
}

func (g getifdef) Apply(ident string, e *Env) ([]string, error) {
	vs, err := e.Get(ident)
	if err == nil {
		vs = []string{g.str}
	}
	return vs, nil
}

func (g getifdef) String() string {
	return fmt.Sprintf("getifdef(%s)", g.str)
}

type identity struct{}

func (i identity) String() string {
	return "identity"
}

func (i identity) Apply(ident string, e *Env) ([]string, error) {
	return e.Get(ident)
}

func Identity() Apply {
	return identity{}
}
