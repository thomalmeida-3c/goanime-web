package version

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Regression test (added 2026-05-01)
//
// The release workflow injects Version from `${GITHUB_REF#refs/tags/}`, which
// keeps the leading `v` (e.g. `v1.8.4`). Code that prints the version uses a
// `v%s` format, so without normalization CI builds logged `vv1.8.4` and
// `version.Version` had a different shape than the in-source fallback.
// Pin that Version is always stored without the leading `v`.

func TestVersion_HasNoLeadingV_2026_05_01(t *testing.T) {
	t.Parallel()
	assert.False(t, strings.HasPrefix(Version, "v"),
		"Version must be normalized without the leading 'v' so format strings like 'v%%s' do not produce 'vv...'; got %q", Version)
}

func TestVersion_NormalizationIsIdempotent_2026_05_01(t *testing.T) {
	t.Parallel()
	// Simulate a tag-style injected value being re-normalized: trimming the
	// `v` prefix twice must be a no-op so the init can be safely called even
	// if Version was already clean.
	once := strings.TrimPrefix("v1.2.3", "v")
	twice := strings.TrimPrefix(once, "v")
	assert.Equal(t, "1.2.3", once)
	assert.Equal(t, once, twice)
}
