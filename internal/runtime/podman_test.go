package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"0B", 0},
		{"100B", 100},
		{"1KB", 1024},
		{"1.5KB", 1536},
		{"100MB", 104857600},
		{"1GB", 1073741824},
		{"1.5GB", 1610612736},
		{"2TB", 2199023255552},
		{"  100MB  ", 104857600}, // with spaces
		{"100mb", 104857600},     // lowercase
	}

	for _, tc := range tests {
		result := parseSize(tc.input)
		assert.Equal(t, tc.expected, result, "Failed for input: %s", tc.input)
	}
}

func TestParseSizeInvalid(t *testing.T) {
	tests := []string{
		"invalid",
		"",
		"abc",
		"100XB",
	}

	for _, input := range tests {
		result := parseSize(input)
		assert.Equal(t, uint64(0), result, "Expected 0 for invalid input: %s", input)
	}
}
