package set

type Set[T comparable] map[T]struct{}

func New[T comparable](elements ...T) Set[T] {
	s := make(Set[T], len(elements))
	s.Add(elements...)
	return s
}

func (s Set[T]) Add(elements ...T) {
	for _, v := range elements {
		s[v] = struct{}{}
	}
}

func (s Set[T]) Del(ele T) {
	delete(s, ele)
}

func (s Set[T]) ToList() []T {
	list := make([]T, 0, len(s))
	for k := range s {
		list = append(list, k)
	}
	return list
}
