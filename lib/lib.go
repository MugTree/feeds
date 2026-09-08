package lib

// Coalesce returns the first non-zero value from vals.
// If all values are the zero value, the zero value is returned.
func Coalesce[T comparable](vals ...T) T {
	var zero T
	for _, v := range vals {
		if v != zero {
			return v
		}
	}
	return zero
}

// Filter returns a new slice containing only the elements
// for which fn returns true.
func Filter[T any](slice []T, fn func(T) bool) []T {
	out := make([]T, 0, len(slice))
	for _, v := range slice {
		if fn(v) {
			out = append(out, v)
		}
	}
	return out
}

// Find returns the first element that satisfies fn.
// The second return value reports whether a match was found.
func Find[T any](slice []T, fn func(T) bool) (T, bool) {
	for _, v := range slice {
		if fn(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// If returns a if cond is true; otherwise it returns b.
func If[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

// IntToBool converts any signed integer type to a bool.
// Zero is false; any non-zero value is true.
func IntToBool[T ~int | ~int8 | ~int16 | ~int32 | ~int64](v T) bool {
	return v != 0
}

// Keys returns a slice containing all keys in m.
// The order of the returned keys is not guaranteed.
func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Map applies fn to every element in slice and returns
// a new slice containing the transformed values.
func Map[T any, R any](slice []T, fn func(T) R) []R {
	out := make([]R, len(slice))
	for i, v := range slice {
		out[i] = fn(v)
	}
	return out
}

// SliceToMap builds a map from items using keyFn to produce
// the key for each element. If duplicate keys are generated,
// the last item wins.
func SliceToMap[T any, K comparable](items []T, keyFn func(T) K) map[K]T {
	m := make(map[K]T, len(items))
	for _, item := range items {
		m[keyFn(item)] = item
	}
	return m
}

// Values returns a slice containing all values in m.
// The order of the returned values is not guaranteed.
func Values[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}
