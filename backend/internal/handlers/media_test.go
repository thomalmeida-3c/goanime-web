package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMediaHandler(t *testing.T) {
	t.Parallel()
	mh := NewMediaHandler()
	require.NotNil(t, mh)
	assert.Equal(t, "Vidcloud", mh.provider)
	assert.Equal(t, "best", mh.quality)
	assert.Equal(t, "english", mh.subsLanguage)
	assert.NotNil(t, mh.mediaManager)
}

func TestMediaHandler_SetOptions(t *testing.T) {
	t.Parallel()
	mh := NewMediaHandler()
	mh.SetOptions("UpCloud", "720p", "spanish")
	assert.Equal(t, "UpCloud", mh.provider)
	assert.Equal(t, "720p", mh.quality)
	assert.Equal(t, "spanish", mh.subsLanguage)

	// Empty values must not overwrite
	mh.SetOptions("", "", "")
	assert.Equal(t, "UpCloud", mh.provider)
	assert.Equal(t, "720p", mh.quality)
	assert.Equal(t, "spanish", mh.subsLanguage)
}

func TestExtractIDFromURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"trailing id", "/movie/watch-foo-12345", "12345"},
		{"single segment", "1234", "1234"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, extractIDFromURL(tt.url))
		})
	}
}

func TestMediaHandler_SelectMedia_EmptyErrors(t *testing.T) {
	t.Parallel()
	mh := NewMediaHandler()
	_, err := mh.SelectMedia(nil)
	assert.Error(t, err)
}
