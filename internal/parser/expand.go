package parser

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/midbel/tish/internal/words"
)

func Expand(str string, args []string, env words.Environment) ([]string, error) {
	var (
		ret []string
		err error
	)
	err = ExpandWith(str, args, env, func(str [][]string) {
		var lines []string
		for i := range str {
			lines = append(lines, strings.Join(str[i], " "))
		}
		ret = append(ret, strings.Join(lines, "; "))
	})
	return ret, err
}

func ExpandWith(str string, args []string, env words.Environment, with func([][]string)) error {
	psr := NewParser(strings.NewReader(str))
	for {
		ex, err := psr.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		str, err := expandExecuter(ex, env)
		if err != nil {
			continue
		}
		with(str)
	}
	return nil
}

func expandExecuter(ex words.Executer, env words.Environment) ([][]string, error) {
	var (
		str [][]string
		err error
	)
	switch x := ex.(type) {
	case words.ExecSimple:
		xs, err1 := x.Expand(env, false)
		if err1 != nil {
			err = err1
			break
		}
		str = append(str, xs)
	case words.ExecAnd:
		left, err1 := expandExecuter(x.Left, env)
		if err1 != nil {
			err = err1
			break
		}
		right, err1 := expandExecuter(x.Right, env)
		if err1 != nil {
			err = err1
			break
		}
		str = append(str, left...)
		str = append(str, right...)
	case words.ExecOr:
		left, err1 := expandExecuter(x.Left, env)
		if err1 != nil {
			err = err1
			break
		}
		right, err1 := expandExecuter(x.Right, env)
		if err1 != nil {
			err = err1
			break
		}
		str = append(str, left...)
		str = append(str, right...)
	case words.ExecPipe:
		for i := range x.List {
			xs, err1 := expandExecuter(x.List[i].Executer, env)
			if err1 != nil {
				err = err1
				break
			}
			str = append(str, xs...)
		}
	default:
		err = fmt.Errorf("unknown/unsupported executer type %T", ex)
	}
	return str, err
}
