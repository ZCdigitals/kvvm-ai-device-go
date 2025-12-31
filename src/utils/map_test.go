package utils

import (
	"strconv"
	"testing"
)

func TestMap(t *testing.T) {
	t.Run("int to string conversion", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		expected := []string{"1", "2", "3", "4", "5"}

		result := Map(input, func(n int) string {
			return strconv.Itoa(n)
		})

		if len(result) != len(expected) {
			t.Errorf("Expected length %d, got %d", len(expected), len(result))
		}

		for i, v := range expected {
			if result[i] != v {
				t.Errorf("At index %d: expected %s, got %s", i, v, result[i])
			}
		}
	})

	t.Run("int to int transformation", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		expected := []int{2, 4, 6, 8, 10}

		result := Map(input, func(n int) int {
			return n * 2
		})

		for i, v := range expected {
			if result[i] != v {
				t.Errorf("At index %d: expected %d, got %d", i, v, result[i])
			}
		}
	})

	t.Run("string transformation", func(t *testing.T) {
		input := []string{"hello", "world", "go"}
		expected := []string{"HELLO", "WORLD", "GO"}

		result := Map(input, func(s string) string {
			return string(Map([]byte(s), func(b byte) byte {
				if b >= 'a' && b <= 'z' {
					return b - 'a' + 'A'
				}
				return b
			}))
		})

		for i, v := range expected {
			if result[i] != v {
				t.Errorf("At index %d: expected %s, got %s", i, v, result[i])
			}
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		input := []int{}

		result := Map(input, func(n int) int {
			return n * 2
		})

		if len(result) != 0 {
			t.Errorf("Expected empty slice, got length %d", len(result))
		}
	})

	t.Run("struct transformation", func(t *testing.T) {
		type Person struct {
			Name string
			Age  int
		}

		input := []Person{
			{"Alice", 25},
			{"Bob", 30},
		}

		result := Map(input, func(p Person) string {
			return p.Name
		})

		expected := []string{"Alice", "Bob"}
		for i, v := range expected {
			if result[i] != v {
				t.Errorf("At index %d: expected %s, got %s", i, v, result[i])
			}
		}
	})
}

func BenchmarkMap(b *testing.B) {
	input := make([]int, 1000)
	for i := range input {
		input[i] = i
	}

	for b.Loop() {
		Map(input, func(n int) int {
			return n * 2
		})
	}
}
