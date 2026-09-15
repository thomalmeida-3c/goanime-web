package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsImageExtension(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"png", ".png", true},
		{"jpg", ".jpg", true},
		{"jpeg", ".jpeg", true},
		{"gif", ".gif", true},
		{"bmp", ".bmp", true},
		{"tiff", ".tiff", true},
		{"webp", ".webp", true},
		{"mp4 rejected", ".mp4", false},
		{"mkv rejected", ".mkv", false},
		{"empty rejected", "", false},
		{"case sensitive — uppercase rejected", ".PNG", false},
		{"missing leading dot rejected", "png", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isImageExtension(tt.in))
		})
	}
}
