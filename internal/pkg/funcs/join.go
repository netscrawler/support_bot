package funcs

func MapJoin[M map[K]V, K comparable, V any](m1 M, m2 M) M {
	result := make(M)
	for k, v := range m1 {
		result[k] = v
	}
	for k, v := range m2 {
		result[k] = v
	}
	return result
}
