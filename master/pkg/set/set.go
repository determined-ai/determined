package set

type unit = struct{}

// Set is an unordered set of values of type T.
type Set[T comparable] map[T]unit

// FromKeys builds a set from the keys of a map.
func FromKeys[M ~map[K]V, K comparable, V any](m M) Set[K] {
	set := make(Set[K], len(m))
	for key := range m {
		set.Insert(key)
	}
	return set
}

// Making Set a defined type rather than a struct means we need the casting shenanigans below, but
// it also allows normal indexing and iteration syntax to be used.

func (s *Set[T]) Contains(val T) bool {
	_, ok := (map[T]unit)(*s)[val]
	return ok
}

func (s *Set[T]) Insert(val T) {
	(map[T]unit)(*s)[val] = unit{}
}

func (s *Set[T]) Copy() Set[T] {
	copy := Set[T]{}
	for x := range (map[T]unit)(*s) {
		copy.Insert(x)
	}
	return copy
}
