package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYtdlpCanImpersonate_DoesNotPanic(t *testing.T) {
	t.Parallel()
	// Result varies by environment; verify the call is safe.
	_ = YtdlpCanImpersonate()
	assert.True(t, true)
}
