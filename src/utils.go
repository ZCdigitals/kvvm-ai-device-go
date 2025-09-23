package src

// Map 函数使用泛型，对切片中的每个元素执行函数f，并返回结果切片
func Map[T any, U any](s []T, f func(T) U) []U {
	result := make([]U, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}
