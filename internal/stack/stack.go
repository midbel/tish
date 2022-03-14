package stack

type Stack[T any] struct {
	list []T
}

func New[T any]() Stack[T] {
	var stk Stack[T]
	return stk
}

func (s *Stack[T]) RotateLeft(n int) {
	if n >= s.Len() {
		return
	}
	s.rotate(n)
}

func (s *Stack[T]) RotateRight(n int) {
	if n >= s.Len() {
		return
	}
	s.rotate(s.Len() - n)
}

func (s *Stack[T]) rotate(n int) {
	s.list = append(s.list[n:], s.list[:n]...)
}

func (s *Stack[T]) Len() int {
	return len(s.list)
}

func (s *Stack[T]) Pop() {
	n := s.Len() - 1
	if n < 0 {
		return
	}
	s.list = s.list[:n]
}

func (s *Stack[T]) RemoveRight(n int) {
	if n < 0 || n >= s.Len() {
		return
	}
	s.list = append(s.list[:n], s.list[n+1:]...)
}

func (s *Stack[T]) RemoveLeft(n int) {
	if n < 0 || n >= s.Len() {
		return
	}
	n = s.Len() - n
	s.list = append(s.list[:n], s.list[n+1:]...)
}

func (s *Stack[T]) Push(item T) {
	s.list = append(s.list, item)
}

func (s *Stack[T]) At(n int) T {
	var ret T
	if n >= s.Len() {
		return ret
	}
	return s.list[n]
}

func (s *Stack[T]) Curr() T {
	var ret T
	if s.Len() > 0 {
		ret = s.list[s.Len()-1]
	}
	return ret
}
