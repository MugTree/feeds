package app

func libSliceToMap[T any, K comparable](items []T, keyFn func(T) K) map[K]T {
	m := make(map[K]T, len(items))
	for _, item := range items {
		m[keyFn(item)] = item
	}
	return m
}
