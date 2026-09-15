package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTMDBMedia_GetDisplayTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		m    *TMDBMedia
		want string
	}{
		{"movie title", &TMDBMedia{Title: "Inception"}, "Inception"},
		{"tv name fallback", &TMDBMedia{Name: "Breaking Bad"}, "Breaking Bad"},
		{"both prefers title", &TMDBMedia{Title: "T", Name: "N"}, "T"},
		{"empty", &TMDBMedia{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.m.GetDisplayTitle())
		})
	}
}

func TestTMDBMedia_GetReleaseYear(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		m    *TMDBMedia
		want string
	}{
		{"movie release date", &TMDBMedia{ReleaseDate: "2010-07-16"}, "2010"},
		{"tv first air date", &TMDBMedia{FirstAirDate: "2008-01-20"}, "2008"},
		{"empty both", &TMDBMedia{}, ""},
		{"short date", &TMDBMedia{ReleaseDate: "201"}, ""},
		{"prefer release over air", &TMDBMedia{ReleaseDate: "2010-01-01", FirstAirDate: "2008-01-01"}, "2010"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.m.GetReleaseYear())
		})
	}
}

func TestTMDBMedia_GetPosterURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		m    *TMDBMedia
		size string
		want string
	}{
		{"with path default size", &TMDBMedia{PosterPath: "/abc.jpg"}, "", "https://image.tmdb.org/t/p/w500/abc.jpg"},
		{"custom size", &TMDBMedia{PosterPath: "/abc.jpg"}, "w780", "https://image.tmdb.org/t/p/w780/abc.jpg"},
		{"empty path", &TMDBMedia{}, "w500", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.m.GetPosterURL(tt.size))
		})
	}
}

func TestTMDBMedia_GetBackdropURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		m    *TMDBMedia
		size string
		want string
	}{
		{"with path default size", &TMDBMedia{BackdropPath: "/bg.jpg"}, "", "https://image.tmdb.org/t/p/w1280/bg.jpg"},
		{"custom size", &TMDBMedia{BackdropPath: "/bg.jpg"}, "original", "https://image.tmdb.org/t/p/original/bg.jpg"},
		{"empty path", &TMDBMedia{}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.m.GetBackdropURL(tt.size))
		})
	}
}
