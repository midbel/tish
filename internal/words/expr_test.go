package words_test

import (
	"testing"

	"github.com/midbel/tish"
)

func TestExpr(t *testing.T) {
	data := []struct {
		tish.Expr
		Want float64
	}{
		{
			Expr: createNumber("1"),
			Want: 1,
		},
		{
			Expr: createUnary(createNumber("1"), tish.Sub),
			Want: -1,
		},
		{
			Expr: createUnary(createNumber("0"), tish.Inc),
			Want: 1,
		},
		{
			Expr: createUnary(createNumber("0"), tish.Dec),
			Want: -1,
		},
		{
			Expr: createBinary(createNumber("1"), createNumber("1"), tish.Mul),
			Want: 1,
		},
		{
			Expr: createBinary(createVariable("sum1"), createVariable("sum2"), tish.Add),
			Want: 2,
		},
	}
	env := tish.EmptyEnv()
	env.Define("sum1", []string{"1"})
	env.Define("sum2", []string{"1"})
	for _, d := range data {
		got, err := d.Expr.Eval(env)
		if err != nil {
			t.Errorf("unexpected error! %s", err)
			continue
		}
		if d.Want != got {
			t.Errorf("results mismatched! want %.2f, got %.2f", d.Want, got)
		}
	}
}

func createNumber(str string) tish.Expr {
	return tish.Number{
		Literal: str,
	}
}

func createUnary(ex tish.Expr, op rune) tish.Expr {
	return tish.Unary{
		Op:   op,
		Expr: ex,
	}
}

func createBinary(left, right tish.Expr, op rune) tish.Expr {
	return tish.Binary{
		Left:  left,
		Right: right,
		Op:    op,
	}
}

func createVariable(ident string) tish.Expr {
	return tish.ExpandVar{
		Ident: ident,
	}
}
