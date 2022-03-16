package tish

import (
	"github.com/midbel/tish/internal/stack"
)

type History interface{}

type history struct {
	list stack.Stack[entry]
}

func HistoryStack() History {
	return &history{
		list: stack.New[entry](),
	}
}

type entry struct {
	name string
	args []string
}
