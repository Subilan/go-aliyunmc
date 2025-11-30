package helpers

// Source: https://github.com/juliangruber/go-intersect/blob/master/intersect.go

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
