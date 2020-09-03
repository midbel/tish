package tish

type Command interface {
	Execute() (int, error)
	Equal(Command) bool
}

type Word struct {
	tokens []Token
}

func (w Word) Equal(other Word) bool {
	if len(w.tokens) != len(other.tokens) {
		return false
	}
	for i, t := range w.tokens {
		if !t.Equal(other.tokens[i]) {
			return false
		}
	}
	return true
}

type Simple struct {
	words []Word
}

func (s Simple) Execute() (int, error) {
	return 0, nil
}

func (s Simple) Equal(other Command) bool {
	i, ok := other.(Simple)
	if !ok {
		return false
	}
	if len(s.words) != len(i.words) {
		return false
	}
	for j, w := range s.words {
		if !w.Equal(i.words[j]) {
			return false
		}
	}
	return true
}

type And struct {
	left  Command
	right Command
}

func (a And) Execute() (int, error) {
	e, err := a.left.Execute()
	if e == 0 && err == nil {
		e, err = a.right.Execute()
	}
	return e, err
}

type Or struct {
	left  Command
	right Command
}

func (o Or) Execute() (int, error) {
	e, err := o.left.Execute()
	if e == 0 && err == nil {
		return e, err
	}
	return o.right.Execute()
}
