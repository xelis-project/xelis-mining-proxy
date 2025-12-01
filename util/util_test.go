package util

import (
	"testing"
)

func TestParseAlgorithm(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"xel/v1", "xel/0"},
		{"xel/v2", "xel/1"},
		{"xel/v3", "xel/2"},
		{"invalid_format", "xel/0"}, // should default to xel/0 and log a warning
	}

	for _, test := range tests {
		result := AlgorithmNodeToStratum(test.input)
		if result != test.expected {
			t.Errorf("AlgorithmNodeToStratum(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}
