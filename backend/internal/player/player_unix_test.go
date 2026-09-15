//go:build !windows

package player

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindMPVPath_ReturnsPathOrErrNotFound(t *testing.T) {
	t.Parallel()
	got, err := findMPVPath()
	if err != nil {
		assert.ErrorIs(t, err, exec.ErrNotFound)
		assert.Empty(t, got)
		return
	}
	// CI environment may or may not have mpv installed; just assert the
	// returned path is non-empty when the function reports success.
	assert.NotEmpty(t, got)
}

func TestSetProcessGroup_SetsSetpgid(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("true")
	setProcessGroup(cmd)
	if assert.NotNil(t, cmd.SysProcAttr) {
		assert.True(t, cmd.SysProcAttr.Setpgid)
	}
}
