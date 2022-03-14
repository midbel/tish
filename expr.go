package tish

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

var ErrZero = errors.New("division by zero")

type Expr interface {
	Eval(Environment) (float64, error)
}

type Number struct {
	Literal string
}

func createNumber(str string) Expr {
	return Number{
		Literal: str,
	}
}

func (n Number) Eval(_ Environment) (float64, error) {
	return strconv.ParseFloat(n.Literal, 64)
}

type Unary struct {
	Op rune
	Expr
}

func createUnary(ex Expr, op rune) Expr {
	return Unary{
		Op:   op,
		Expr: ex,
	}
}

func (u Unary) Eval(env Environment) (float64, error) {
	ret, err := u.Expr.Eval(env)
	if err != nil {
		return ret, err
	}
	switch u.Op {
	case Not:
		if ret != 0 {
			ret = 1
		}
	case Sub:
		ret = -ret
	case Inc:
		ret = ret + 1
	case Dec:
		ret = ret - 1
	case BitNot:
		x := ^int64(ret)
		ret = float64(x)
	default:
		return 0, fmt.Errorf("unsupported operator")
	}
	return ret, nil
}

type Binary struct {
	Op    rune
	Left  Expr
	Right Expr
}

func (b Binary) Eval(env Environment) (float64, error) {
	left, err := b.Left.Eval(env)
	if err != nil {
		return left, err
	}
	right, err := b.Right.Eval(env)
	if err != nil {
		return right, err
	}
	do, ok := binaries[b.Op]
	if !ok {
		return 0, fmt.Errorf("unsupported operator")
	}
	return do(left, right)
}

type Ternary struct {
	Cond  Expr
	Left  Expr
	Right Expr
}

func (t Ternary) Eval(env Environment) (float64, error) {
	cdt, err := t.Cond.Eval(env)
	if err != nil {
		return cdt, err
	}
	if cdt == 0 {
		return t.Right.Eval(env)
	}
	return t.Left.Eval(env)
}

type Assignment struct {
	Ident string
	Expr
}

func (a Assignment) Eval(env Environment) (float64, error) {
	ret, err := a.Expr.Eval(env)
	if err != nil {
		return ret, err
	}
	str := strconv.FormatFloat(ret, 'f', -1, 64)
	return ret, env.Define(a.Ident, []string{str})
}

type bind int8

const (
	bindLowest bind = iota
	bindAssign
	bindBit
	bindShift
	bindTernary
	bindLogical
	bindEq
	bindCmp
	bindAdd
	bindMul
	bindPow
	bindPrefix
)

var bindings = map[rune]bind{
	BitAnd:     bindBit,
	BitOr:      bindBit,
	BitXor:     bindBit,
	Add:        bindAdd,
	Sub:        bindAdd,
	Mul:        bindMul,
	Div:        bindMul,
	Mod:        bindMul,
	Pow:        bindPow,
	LeftShift:  bindShift,
	RightShift: bindShift,
	And:        bindLogical,
	Or:         bindLogical,
	Eq:         bindEq,
	Ne:         bindEq,
	Lt:         bindCmp,
	Le:         bindCmp,
	Gt:         bindCmp,
	Ge:         bindCmp,
	SameFile:   bindCmp,
	NewerThan:  bindCmp,
	OlderThan:  bindCmp,
	Cond:       bindTernary,
	Alt:        bindTernary,
	Assign:     bindAssign,
}

func bindPower(tok Token) bind {
	pow, ok := bindings[tok.Type]
	if !ok {
		pow = bindLowest
	}
	return pow
}

var binaries = map[rune]func(float64, float64) (float64, error){
	Add:        doAdd,
	Sub:        doSub,
	Mul:        doMul,
	Div:        doDiv,
	Mod:        doMod,
	Pow:        doPow,
	LeftShift:  doLeft,
	RightShift: doRight,
	Eq:         doEq,
	Ne:         doNe,
	Lt:         doLt,
	Le:         doLe,
	Gt:         doGt,
	Ge:         doGe,
	And:        doAnd,
	Or:         doOr,
	BitAnd:     doBitAnd,
	BitOr:      doBitOr,
	BitXor:     doBitXor,
}

func doAdd(left, right float64) (float64, error) {
	return left + right, nil
}

func doSub(left, right float64) (float64, error) {
	return left - right, nil
}

func doMul(left, right float64) (float64, error) {
	return left * right, nil
}

func doPow(left, right float64) (float64, error) {
	return math.Pow(left, right), nil
}

func doDiv(left, right float64) (float64, error) {
	if right == 0 {
		return right, ErrZero
	}
	return left / right, nil
}

func doMod(left, right float64) (float64, error) {
	if right == 0 {
		return right, ErrZero
	}
	return math.Mod(left, right), nil
}

func doLeft(left, right float64) (float64, error) {
	if left < 0 {
		return 0, nil
	}
	x := int64(left) << int64(right)
	return float64(x), nil
}

func doRight(left, right float64) (float64, error) {
	if left < 0 {
		return 0, nil
	}
	x := int64(left) >> int64(right)
	return float64(x), nil
}

func doEq(left, right float64) (float64, error) {
	if left == right {
		return 1, nil
	}
	return 0, nil
}

func doNe(left, right float64) (float64, error) {
	if left != right {
		return 1, nil
	}
	return 0, nil
}

func doLt(left, right float64) (float64, error) {
	if left < right {
		return 1, nil
	}
	return 0, nil
}

func doLe(left, right float64) (float64, error) {
	if left <= right {
		return 1, nil
	}
	return 0, nil
}

func doGt(left, right float64) (float64, error) {
	if left > right {
		return 1, nil
	}
	return 0, nil
}

func doGe(left, right float64) (float64, error) {
	if left >= right {
		return 1, nil
	}
	return 0, nil
}

func doAnd(left, right float64) (float64, error) {
	if left == 0 && right == 0 {
		return left, nil
	}
	return 1, nil
}

func doOr(left, right float64) (float64, error) {
	if left == 0 || right == 0 {
		return 0, nil
	}
	return 1, nil
}

func doBitAnd(left, right float64) (float64, error) {
	x := int64(left) & int64(right)
	return float64(x), nil
}

func doBitOr(left, right float64) (float64, error) {
	x := int64(left) | int64(right)
	return float64(x), nil
}

func doBitXor(left, right float64) (float64, error) {
	x := int64(left) ^ int64(right)
	return float64(x), nil
}
