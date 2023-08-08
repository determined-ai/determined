package set

type unit = struct{}

// Set is an unordered set of values of type T.
type Set[T comparable] map[T]unit

// Making Set a defined type rather than a struct means we need the casting shenanigans below, but
// it also allows normal indexing and iteration syntax to be used.

// New returns a empty set.
func New[T comparable]() Set[T] {
	return make(Set[T])
}

// FromSlice returns a set containing the values in the given slice.
func FromSlice[T comparable](keys []T) Set[T] {
	set := make(Set[T])
	for _, x := range keys {
		set.Insert(x)
	}
	return set
}

// FromKeys builds a set from the keys of a map.
func FromKeys[M ~map[K]V, K comparable, V any](m M) Set[K] {
	set := make(Set[K], len(m))
	for key := range m {
		set.Insert(key)
	}
	return set
}

// Contains checks whether the passed-in value is present in the Set.
func (s *Set[T]) Contains(val T) bool {
	_, ok := (map[T]unit)(*s)[val]
	return ok
}

// Insert adds the passed-in value to the Set.
func (s *Set[T]) Insert(val T) {
	(map[T]unit)(*s)[val] = unit{}
}

// Remove removes the passed-in value from the Set.
func (s *Set[T]) Remove(val T) {
	delete((map[T]unit)(*s), val)
}

// ToSlice builds a new slice, populates it with the contents of the Set, and returns it.
func (s Set[T]) ToSlice() []T {
	res := make([]T, 0, len(s))
	for val := range s {
		res = append(res, val)
	}
	return res
}
