package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToTitleCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"single word", "naruto", "Naruto"},
		{"already title", "Naruto", "Naruto"},
		{"multi word", "nanatsu no taizai", "Nanatsu No Taizai"},
		{"mixed case preserves rest", "nARUTo", "NARUTo"},
		{"extra spaces collapse", "  shingeki   no   kyojin  ", "Shingeki No Kyojin"},
		{"single letter word", "a b c", "A B C"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, toTitleCase(tt.in))
		})
	}
}
