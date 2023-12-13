package tish

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"
)

var (
	ErrExpansion = errors.New("expansion too many words")
	ErrZero      = errors.New("division by zero")
)

type Expr interface {
	Word
	Test(Environment) (bool, error)
	Eval(Environment) (float64, error)
}

type Word interface {
	Expand(Environment) ([]string, error)
}

type number float64

func createNumber(n float64) Word {
	return number(n)
}

func (n number) Expand(_ Environment) ([]string, error) {
	x := strconv.FormatFloat(float64(n), 'f', -1, 64)
	return strArray(x), nil
}

func (n number) Test(_ Environment) (bool, error) {
	return float64(n) != 0, nil
}

func (n number) Eval(_ Environment) (float64, error) {
	return float64(n), nil
}

type literal string

func createLiteral(str string) Word {
	return literal(str)
}

func (i literal) Expand(_ Environment) ([]string, error) {
	vs := []string{string(i)}
	return vs, nil
}

func (i literal) Test(_ Environment) (bool, error) {
	ok := len(i) > 0
	return ok, nil
}

func (i literal) Eval(env Environment) (float64, error) {
	list, err := i.Expand(env)
	if err != nil {
		return 0, err
	}
	if len(list) != 1 {
		return 0, ErrExpansion
	}
	return strconv.ParseFloat(list[0], 64)
}

type identifier string

func createIdentifier(ident string) Word {
	return identifier(ident)
}

func (i identifier) Expand(env Environment) ([]string, error) {
	return env.Resolve(string(i))
}

func (i identifier) Test(env Environment) (bool, error) {
	_, err := env.Resolve(string(i))
	return err == nil, nil
}

func (i identifier) Eval(env Environment) (float64, error) {
	list, err := i.Expand(env)
	if err != nil {
		return 0, err
	}
	if len(list) != 1 {
		return 0, ErrExpansion
	}
	return strconv.ParseFloat(list[0], 64)
}

type combined []Word

func createCombined(list []Word) Word {
	if len(list) == 1 {
		return list[0]
	}
	return combined(list)
}

func (c combined) Expand(env Environment) ([]string, error) {
	var list []string
	for i := range c {
		str, err := c[i].Expand(env)
		if err != nil {
			return nil, err
		}
		list = append(list, str...)
	}
	str := strings.Join(list, "")
	return strArray(str), nil
}

func (c combined) Test(env Environment) (bool, error) {
	for _, w := range c {
		t, ok := w.(Expr)
		if !ok {
			return false, nil
		}
		if ok, _ = t.Test(env); !ok {
			return ok, nil
		}
	}
	return true, nil
}

type substitution string

func createSubstitution(str string) Word {
	return substitution(str)
}

func (s substitution) Expand(env Environment) ([]string, error) {
	var (
		sh *Shell
		err error
	)
	r := strings.NewReader(string(s))
	if sub, ok := env.(interface{ SubShell(io.Reader) (*Shell, error) }); ok {
		sh, err = sub.SubShell(r)
	} else {
		sh, err = NewShellWithEnv(r, env)
	}
	if err != nil {
		return nil, err
	}
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	sh.Stdout = &stdout
	sh.Stderr = &stderr
	if err := sh.Run(); err != nil {
		return nil, fmt.Errorf("%s: %s", err, stderr.String())
	}
	str := strings.TrimSpace(stdout.String())
	return strArray(str), nil
}

type expr struct {
	Expr
}

func createExpr(e Expr) Word {
	return expr{
		Expr: e,
	}
}

func (e expr) Expand(env Environment) ([]string, error) {
	n, err := e.Expr.Eval(env)
	if err != nil {
		return nil, err
	}
	x := strconv.FormatFloat(n, 'f', -1, 64)
	return strArray(x), nil
}

type stdRedirect struct {
	Word
	Kind rune
}

func redirectReader(w Word) Word {
	return stdRedirect{
		Word: w,
		Kind: RedirectIn,
	}
}

func redirectWriter(w Word, kind rune) Word {
	return stdRedirect{
		Word: w,
		Kind: kind,
	}
}

func appendWriter(w Word, kind rune) Word {
	return stdRedirect{
		Word: w,
		Kind: kind,
	}
}

type trimmer interface {
	trim(string) ([]string, error)
}

type trim struct {
	ident string
	trimmer
}

func (r trim) Expand(env Environment) ([]string, error) {
	str, err := env.Resolve(r.ident)
	if err != nil {
		return nil, err
	}
	if len(str) != 1 {
		return nil, ErrExpansion
	}
	return r.trim(str[0])
}

type removePrefix struct {
	long bool
	str  string
}

func getRemovePrefix(str string, long bool) trimmer {
	return removePrefix{
		long: long,
		str:  str,
	}
}

func (r removePrefix) trim(str string) ([]string, error) {
	str = strings.TrimPrefix(str, r.str)
	for strings.HasPrefix(str, r.str) && r.long {
		str = strings.TrimPrefix(str, r.str)
	}
	return strArray(str), nil
}

type removeSuffix struct {
	long bool
	str  string
}

func getRemoveSuffix(str string, long bool) trimmer {
	return removeSuffix{
		long: long,
		str:  str,
	}
}

func (r removeSuffix) trim(str string) ([]string, error) {
	str = strings.TrimSuffix(str, r.str)
	for strings.HasSuffix(str, r.str) && r.long {
		str = strings.TrimSuffix(str, r.str)
	}
	return strArray(str), nil
}

type transformer interface {
	transform(string) ([]string, error)
}

type caser struct {
	ident string
	transformer
}

func (c caser) Expand(env Environment) ([]string, error) {
	str, err := env.Resolve(c.ident)
	if err != nil {
		return c.transform(c.ident)
	}
	if len(str) != 1 {
		return nil, ErrExpansion
	}
	return c.transform(str[0])
}

type lowerCase struct {
	all bool
}

func getLowercaseTransformer(all bool) transformer {
	return lowerCase{
		all: all,
	}
}

func (c lowerCase) transform(str string) ([]string, error) {
	if len(str) == 0 {
		return strArray(str), nil
	}
	if c.all {
		str = strings.ToLower(str)
	} else {
		str = transformCase(str, unicode.ToLower)
	}
	return strArray(str), nil
}

type upperCase struct {
	all bool
}

func getUppercaseTransformer(all bool) transformer {
	return upperCase{
		all: all,
	}
}

func (c upperCase) transform(str string) ([]string, error) {
	if len(str) == 0 {
		return strArray(str), nil
	}
	if c.all {
		str = strings.ToUpper(str)
	} else {
		str = transformCase(str, unicode.ToUpper)
	}
	return strArray(str), nil
}

func transformCase(str string, do func(rune) rune) string {
	if len(str) == 0 {
		return str
	}
	raw := []rune(str)
	raw[0] = do(raw[0])
	return string(raw)
}

type replacer interface {
	replace(str string) ([]string, error)
}

type replace struct {
	ident string
	replacer
}

func (r replace) Expand(env Environment) ([]string, error) {
	str, err := env.Resolve(r.ident)
	if err != nil {
		return nil, err
	}
	if len(str) != 1 {
		return nil, ErrExpansion
	}
	return r.replace(str[0])
}

type replaceOne struct {
	old string
	new string
}

func getReplaceOne(old, new string) replacer {
	return replaceOne{
		old: old,
		new: new,
	}
}

func (r replaceOne) replace(str string) ([]string, error) {
	str = strings.Replace(str, r.old, r.new, 1)
	return strArray(str), nil
}

type replaceAll struct {
	old string
	new string
}

func getReplaceAll(old, new string) replacer {
	return replaceAll{
		old: old,
		new: new,
	}
}

func (r replaceAll) replace(str string) ([]string, error) {
	str = strings.ReplaceAll(str, r.old, r.new)
	return strArray(str), nil
}

type replacePrefix struct {
	old string
	new string
}

func getReplacePrefix(old, new string) replacer {
	return replacePrefix{
		old: old,
		new: new,
	}
}

func (r replacePrefix) replace(str string) ([]string, error) {
	str, ok := strings.CutPrefix(str, r.old)
	if ok {
		str = r.new + str
	}
	return strArray(str), nil
}

type replaceSuffix struct {
	old string
	new string
}

func getReplaceSuffix(old, new string) replacer {
	return replaceSuffix{
		old: old,
		new: new,
	}
}

func (r replaceSuffix) replace(str string) ([]string, error) {
	str, ok := strings.CutSuffix(str, r.old)
	if ok {
		str = str + r.new
	}
	return strArray(str), nil
}

type substring struct {
	ident  string
	offset int
	length int
}

func (s substring) Expand(env Environment) ([]string, error) {
	str, err := env.Resolve(s.ident)
	if err != nil {
		return nil, err
	}
	if len(str) != 1 {
		return nil, ErrExpansion
	}
	return s.substring(str[0])
}

func (s substring) substring(str string) ([]string, error) {
	if len(str) == 0 || s.length == 0 {
		return strArray(""), nil
	}
	var (
		offset = s.offset
		end    = len(str)
	)
	if offset < 0 {
		offset = end + s.offset
	}
	if s.length < 0 {
		end += s.length
	} else {
		end = s.offset + s.length
	}
	if offset >= end {
		str = ""
	} else {
		str = str[offset:end]
	}
	return strArray(str), nil
}

type length struct {
	ident string
}

func (e length) Expand(env Environment) ([]string, error) {
	str, err := env.Resolve(e.ident)
	if err != nil {
		return nil, err
	}
	if len(str) != 1 {
		return nil, ErrExpansion
	}
	n := strconv.Itoa(len(str[0]))
	return strArray(n), nil
}

type unaryExpr struct {
	op   string
	word Expr
}

func (u unaryExpr) Eval(env Environment) (float64, error) {
	right, err := u.word.Eval(env)
	if err != nil {
		return right, err
	}
	switch u.op {
	case "-":
		right = -right
	case "!":
		if right != 0 {
			right = 0
		}
	default:
		return 0, fmt.Errorf("eval: unsupported unary operator: %s", u.op)
	}
	return right, nil
}

func (u unaryExpr) Test(env Environment) (bool, error) {
	if u.op == "!" {
		ok, err := u.word.Test(env)
		return !ok, err
	}
	list, err := u.word.Expand(env)
	if err != nil {
		return false, err
	}
	if len(list) != 1 {
		return false, ErrExpansion
	}
	switch str := list[0]; u.op {
	case "e":
		return testFileExists(str)
	case "r":
		return testFileReadable(str)
	case "h":
		return testFileSymlink(str)
	case "d":
		return testFileDirectory(str)
	case "w":
		return testFileWritable(str)
	case "s":
		return testFileSize(str)
	case "f":
		return testFileRegular(str)
	case "x":
		return testFileExecutable(str)
	case "z":
		return testStringEmpty(str)
	case "n":
		return testStringNotEmpty(str)
	default:
		return false, fmt.Errorf("test: unsupported unary operator: %s", u.op)
	}
}

func (_ unaryExpr) Expand(_ Environment) ([]string, error) {
	return nil, fmt.Errorf("'unary' can not be expanded")
}

type binaryExpr struct {
	op    string
	left  Expr
	right Expr
}

func (b binaryExpr) Eval(env Environment) (float64, error) {
	left, err := b.left.Eval(env)
	if err != nil {
		return 0, err
	}
	right, err := b.right.Eval(env)
	if err != nil {
		return 0, err
	}
	switch b.op {
	case "+":
		left += right
	case "-":
		left -= right
	case "*":
		left *= right
	case "/":
		if right == 0 {
			return 0, ErrZero
		}
		left /= right
	case "%":
		if right == 0 {
			return 0, ErrZero
		}
		left = math.Mod(left, right)
	case "==":
		if left == right {
			left = 0
		}
	case "!=":
		if left != right {
			left = 0
		}
	case "<":
		if left < right {
			left = 0
		}
	case "<=":
		if left <= right {
			left = 0
		}
	case ">":
		if left > right {
			left = 0
		}
	case ">=":
		if left >= right {
			left = 0
		}
	case "<<":
		x := int64(left) << int64(right)
		left = float64(x)
	case ">>":
		x := int64(left) >> int64(right)
		left = float64(x)
	case "&":
		x := int64(left) & int64(right)
		left = float64(x)
	case "|":
		x := int64(left) | int64(right)
		left = float64(x)
	case "&&":
		if left == 0 && right == 0 {
			left = 0
		}
	case "||":
		if left == 0 || right == 0 {
			left = 0
		}
	default:
		return 0, fmt.Errorf("eval: unsupported binary operator: %s", b.op)
	}
	return left, nil
}

func (b binaryExpr) Test(env Environment) (bool, error) {
	if b.op == "&&" || b.op == "||" {
		return b.testLogical(env)
	}
	ls, err := b.left.Expand(env)
	if err != nil {
		return false, err
	}
	rs, err := b.right.Expand(env)
	if err != nil {
		return false, err
	}
	if len(ls) != 1 || len(rs) != 1 {
		return false, fmt.Errorf("invalid number of words")
	}
	switch left, right := ls[0], rs[0]; b.op {
	case "==":
		return testStringEqual(left, right)
	case "!=":
		return testStringNotEqual(left, right)
	case "eq":
		return testNumberEqual(left, right)
	case "ne":
		return testNumberNotEqual(left, right)
	case "lt":
		return testNumberLessThan(left, right)
	case "le":
		return testNumberLessEqual(left, right)
	case "gt":
		return testNumberGreatThan(left, right)
	case "ge":
		return testNumberGreatEqual(left, right)
	case "nt":
		return testFileNewerThan(left, right)
	case "ot":
		return testFileOlderThan(left, right)
	case "ef":
		return testFileSameFile(left, right)
	default:
		return false, fmt.Errorf("test: unsupported binary operator: %s", b.op)
	}
	return false, nil
}

func (b binaryExpr) testLogical(env Environment) (bool, error) {
	left, err := b.left.Test(env)
	if err != nil {
		return false, err
	}
	right, err := b.right.Test(env)
	if err != nil {
		return false, err
	}
	if b.op == "&&" {
		return left && right, nil
	}
	return left || right, nil
}

func (_ binaryExpr) Expand(_ Environment) ([]string, error) {
	return nil, fmt.Errorf("'binary test' can not be expanded")
}

func testFileExists(file string) (bool, error) {
	return statFile(file, func(_ os.FileInfo) bool { return true })
}

func testFileReadable(file string) (bool, error) {
	return statFile(file, func(fi os.FileInfo) bool {
		var (
			perm  = fi.Mode().Perm()
			owner = perm&0600 != 0
			group = perm&0060 != 0
			other = perm&0006 != 0
		)
		return owner || group || other
	})
}

func testFileSymlink(file string) (bool, error) {
	return statFile(file, func(fi os.FileInfo) bool {
		return fi.Mode()&os.ModeSymlink == os.ModeSymlink
	})
}

func testFileDirectory(file string) (bool, error) {
	return statFile(file, func(fi os.FileInfo) bool {
		return fi.IsDir()
	})
}

func testFileWritable(file string) (bool, error) {
	return statFile(file, func(fi os.FileInfo) bool {
		var (
			perm  = fi.Mode().Perm()
			owner = perm&0400 != 0
			group = perm&0040 != 0
			other = perm&0004 != 0
		)
		return owner || group || other
	})
}

func testFileSize(file string) (bool, error) {
	return statFile(file, func(fi os.FileInfo) bool {
		return fi.Size() > 0
	})
}

func testFileRegular(file string) (bool, error) {
	return statFile(file, func(fi os.FileInfo) bool {
		return fi.Mode().IsRegular()
	})
}

func testFileExecutable(file string) (bool, error) {
	return statFile(file, func(fi os.FileInfo) bool {
		var (
			perm  = fi.Mode().Perm()
			owner = perm&0100 != 0
			group = perm&0010 != 0
			other = perm&0001 != 0
		)
		return owner || group || other
	})
}

func testFileNewerThan(file1, file2 string) (bool, error) {
	f1, err1 := os.Stat(file1)
	f2, err2 := os.Stat(file2)
	if err1 == nil && err2 != nil {
		return true, nil
	}
	return f1.ModTime().After(f2.ModTime()), nil
}

func testFileOlderThan(file1, file2 string) (bool, error) {
	f1, err1 := os.Stat(file1)
	f2, err2 := os.Stat(file2)
	if err1 != nil && err2 == nil {
		return true, nil
	}
	return f1.ModTime().Before(f2.ModTime()), nil
}

func testFileSameFile(file1, file2 string) (bool, error) {
	f1, err1 := os.Stat(file1)
	f2, err2 := os.Stat(file2)
	if err1 != nil || err2 != nil {
		return false, nil
	}
	return os.SameFile(f1, f2), nil
}

func testStringEmpty(str string) (bool, error) {
	return str == "", nil
}

func testStringNotEmpty(str string) (bool, error) {
	return str != "", nil
}

func testStringEqual(str1, str2 string) (bool, error) {
	return str1 == str2, nil
}

func testStringNotEqual(str1, str2 string) (bool, error) {
	return str1 != str2, nil
}

func testNumberEqual(str1, str2 string) (bool, error) {
	return cmpNumber(str1, str2, func(n1, n2 float64) bool {
		return n1 == n2
	})
}

func testNumberNotEqual(str1, str2 string) (bool, error) {
	ok, err := testNumberEqual(str1, str2)
	if err == nil {
		ok = !ok
	}
	return ok, err
}

func testNumberLessThan(str1, str2 string) (bool, error) {
	return cmpNumber(str1, str2, func(n1, n2 float64) bool {
		return n1 < n2
	})
}

func testNumberLessEqual(str1, str2 string) (bool, error) {
	return cmpNumber(str1, str2, func(n1, n2 float64) bool {
		return n1 <= n2
	})
}

func testNumberGreatThan(str1, str2 string) (bool, error) {
	return cmpNumber(str1, str2, func(n1, n2 float64) bool {
		return n1 > n2
	})
}

func testNumberGreatEqual(str1, str2 string) (bool, error) {
	return cmpNumber(str1, str2, func(n1, n2 float64) bool {
		return n1 >= n2
	})
}

func statFile(file string, check func(os.FileInfo) bool) (bool, error) {
	s, err := os.Stat(file)
	if err != nil {
		return false, err
	}
	return check(s), nil
}

func cmpNumber(str1, str2 string, check func(float64, float64) bool) (bool, error) {
	n1, err := strconv.ParseFloat(str1, 64)
	if err != nil {
		return false, err
	}
	n2, err := strconv.ParseFloat(str2, 64)
	if err != nil {
		return false, err
	}
	return check(n1, n2), nil
}
