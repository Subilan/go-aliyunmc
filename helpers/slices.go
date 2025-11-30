package helpers

import "sort"

// Source: https://github.com/juliangruber/go-intersect/blob/master/intersect.go

// IntersectSortedGeneric has complexity: O(n * log(n)), a needs to be sorted
func IntersectSortedGeneric[T comparable](a []T, b []T) []T {
	set := make([]T, 0)

	for _, v := range a {
		idx := sort.Search(len(b), func(i int) bool {
			return b[i] == v
		})
		if idx < len(b) && b[idx] == v {
			set = append(set, v)
		}
	}

	return set
}

// IntersectHashGeneric has complexity: O(n * x) where x is a factor of hash function efficiency (between 1 and 2)
func IntersectHashGeneric[T comparable](a []T, b []T) []T {
	set := make([]T, 0)
	hash := make(map[T]struct{})

	for _, v := range a {
		hash[v] = struct{}{}
	}

	for _, v := range b {
		if _, ok := hash[v]; ok {
			set = append(set, v)
		}
	}

	return set
}
