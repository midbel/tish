package words_test

import (
	"testing"

	"github.com/midbel/tish"
	"github.com/midbel/tish/internal/token"
	"github.com/midbel/tish/internal/words"
)

func TestExpr(t *testing.T) {
	data := []struct {
		words.Expr
		Want float64
	}{
		{
			Expr: createNumber("1"),
			Want: 1,
		},
		{
			Expr: createUnary(createNumber("1"), token.Sub),
			Want: -1,
		},
		{
			Expr: createUnary(createNumber("0"), token.Inc),
			Want: 1,
		},
		{
			Expr: createUnary(createNumber("0"), token.Dec),
			Want: -1,
		},
		{
			Expr: createBinary(createNumber("1"), createNumber("1"), token.Mul),
			Want: 1,
		},
		{
			Expr: createBinary(createVariable("sum1"), createVariable("sum2"), token.Add),
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

func createNumber(str string) words.Expr {
	return words.Number{
		Literal: str,
	}
}

func createUnary(ex words.Expr, op rune) words.Expr {
	return words.Unary{
		Op:   op,
		Expr: ex,
	}
}

func createBinary(left, right words.Expr, op rune) words.Expr {
	return words.Binary{
		Left:  left,
		Right: right,
		Op:    op,
	}
}

func createVariable(ident string) words.Expr {
	return words.ExpandVar{
		Ident: ident,
	}
}
