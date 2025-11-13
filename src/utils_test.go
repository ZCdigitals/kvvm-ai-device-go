// utils_test.go
package src

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

func TestBoolsToInt(t *testing.T) {
	testCases := []struct {
		name     string
		input    []bool
		expected int
	}{
		{
			name:     "single true",
			input:    []bool{true},
			expected: 1,
		},
		{
			name:     "single false",
			input:    []bool{false},
			expected: 0,
		},
		{
			name:     "multiple true values",
			input:    []bool{true, false, true},
			expected: 1<<0 | 1<<2, // 1 + 4 = 5
		},
		{
			name:     "all true",
			input:    []bool{true, true, true},
			expected: 1<<0 | 1<<1 | 1<<2, // 1 + 2 + 4 = 7
		},
		{
			name:     "all false",
			input:    []bool{false, false, false},
			expected: 0,
		},
		{
			name:     "mixed values complex",
			input:    []bool{true, false, false, true, false, true},
			expected: 1<<0 | 1<<3 | 1<<5, // 1 + 8 + 32 = 41
		},
		{
			name:     "empty input",
			input:    []bool{},
			expected: 0,
		},
		{
			name:     "position based - first bit",
			input:    []bool{true, false, false, false},
			expected: 1,
		},
		{
			name:     "position based - second bit",
			input:    []bool{false, true, false, false},
			expected: 2,
		},
		{
			name:     "position based - third bit",
			input:    []bool{false, false, true, false},
			expected: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := BoolsToInt(tc.input...)
			if result != tc.expected {
				t.Errorf("BoolsToInt(%v) = %d; expected %d",
					tc.input, result, tc.expected)
			}
		})
	}
}

// 基准测试
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

func BenchmarkBoolsToInt(b *testing.B) {
	boolSlice := []bool{true, false, true, false, true, true, false, true}

	for b.Loop() {
		BoolsToInt(boolSlice...)
	}
}

// 示例测试
func ExampleMap() {
	numbers := []int{1, 2, 3, 4, 5}
	result := Map(numbers, func(n int) string {
		return strconv.Itoa(n)
	})

	for _, s := range result {
		println(s)
	}
	// Output would be strings "1", "2", "3", "4", "5"
}

func ExampleBoolsToInt() {
	result := BoolsToInt(true, false, true)
	println(result) // 1 (bit 0) + 4 (bit 2) = 5
	// Output: 5
}
