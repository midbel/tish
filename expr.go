package tish

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	bindLowest int = iota
	bindShift
	bindLogical
	bindPlus
	bindMul
	bindPrefix
)

var bindings = map[rune]int{
	plus:          bindPlus,
	minus:         bindPlus,
	mul:           bindMul,
	div:           bindMul,
	modulo:        bindMul,
	tokLeftShift:  bindShift,
	tokRightShift: bindShift,
	ampersand:     bindLogical,
	pipe:          bindLogical,
}

func bindPower(tok Token) int {
	p, ok := bindings[tok.Type]
	if !ok {
		p = bindLowest
	}
	return p
}

type Expr struct {
	expr Evaluator
}

func (e Expr) Expand(env *Env) ([]string, error) {
	n, err := e.expr.Eval(env)
	if err != nil {
		return nil, err
	}
	x := strconv.FormatInt(int64(n), 10)
	return []string{x}, nil
}

func (e Expr) String() string {
	return e.expr.String()
}

func (e Expr) Equal(w Word) bool {
	other, ok := w.(Expr)
	if !ok {
		return ok
	}
	return e.String() == other.String()
}

func (e Expr) asWord() Word {
	return e
}

type Evaluator interface {
	Eval(*Env) (Number, error)
	fmt.Stringer
}

type infix struct {
	left  Evaluator
	right Evaluator
	op    rune
}

func (i infix) Eval(e *Env) (Number, error) {
	left, err := i.left.Eval(e)
	if err != nil {
		return 0, err
	}
	right, err := i.right.Eval(e)
	if err != nil {
		return 0, err
	}
	var r Number
	switch i.op {
	default:
		return 0, fmt.Errorf("unsupported infix operator: %c", i.op)
	case plus:
		r = left + right
	case minus:
		r = left - right
	case div:
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		r = left / right
	case mul:
		r = left * right
	case modulo:
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		r = left % right
	case tokLeftShift:
		if right < 0 {
			return 0, fmt.Errorf("negative shift count: %d", right)
		}
		r = left << right
	case tokRightShift:
		if right < 0 {
			return 0, fmt.Errorf("negative shift count: %d", right)
		}
		r = left >> right
	case ampersand:
		r = left & right
	case pipe:
		r = left | right
	}
	return r, nil
}

func (i infix) String() string {
	var buf strings.Builder
	buf.WriteString("infix(")
	buf.WriteString(i.left.String())
	switch i.op {
	case tokLeftShift:
		buf.WriteString("<<")
	case tokRightShift:
		buf.WriteString(">>")
	default:
		buf.WriteRune(i.op)
	}
	buf.WriteString(i.right.String())
	buf.WriteString(")")
	return buf.String()
}

type prefix struct {
	right Evaluator
	op    rune
}

func (p prefix) Eval(e *Env) (Number, error) {
	right, err := p.right.Eval(e)
	if err != nil {
		return 0, err
	}
	switch p.op {
	default:
		return 0, fmt.Errorf("unsupported prefix operator: %c", p.op)
	case minus:
		return -right, nil
	}
}

func (p prefix) String() string {
	var buf strings.Builder
	buf.WriteString("prefix(")
	buf.WriteRune(p.op)
	buf.WriteString(p.right.String())
	buf.WriteString(")")
	return buf.String()
}
