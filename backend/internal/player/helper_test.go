package player

import (
	"testing"
	"time"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_ReturnsBatchCommand(t *testing.T) {
	t.Parallel()
	m := &model{progress: progress.New()}
	cmd := m.Init()
	assert.NotNil(t, cmd, "Init must return a non-nil tea.Cmd")
}

func TestTickCmd_ProducesTickMsg(t *testing.T) {
	t.Parallel()
	cmd := tickCmd()
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(tickMsg)
	assert.True(t, ok, "tickCmd should emit tickMsg, got %T", msg)
}

func TestUpdate_TickMsg_DoneTriggersQuitAfterMaxFrames(t *testing.T) {
	t.Parallel()
	m := &model{
		progress:   progress.New(),
		done:       true,
		totalBytes: 100,
		received:   100,
	}
	// Push enough ticks to exceed the 3-frame quiet window (no err).
	var lastCmd tea.Cmd
	for range 4 {
		_, lastCmd = m.Update(tickMsg(time.Now()))
	}
	require.NotNil(t, lastCmd)
}

func TestUpdate_TickMsg_ProgressUpdatesPeak(t *testing.T) {
	t.Parallel()
	m := &model{
		progress:   progress.New(),
		totalBytes: 100,
		received:   50,
	}
	_, cmd := m.Update(tickMsg(time.Now()))
	assert.NotNil(t, cmd)
	assert.InDelta(t, 0.5, m.peakPct, 1e-9)
}

func TestUpdate_StatusMsgUpdatesStatus(t *testing.T) {
	t.Parallel()
	m := &model{progress: progress.New()}
	_, cmd := m.Update(statusMsg("downloading…"))
	assert.Nil(t, cmd)
	assert.Equal(t, "downloading…", m.status)
}

func TestUpdate_UnknownMsgIsNoop(t *testing.T) {
	t.Parallel()
	m := &model{progress: progress.New(), status: "x"}
	_, cmd := m.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.Equal(t, "x", m.status)
}

func TestView_RendersStatusAndQuitInstruction(t *testing.T) {
	t.Parallel()
	m := &model{
		progress: progress.New(progress.WithDefaultBlend()),
		status:   "Downloading episode 7",
	}
	v := m.View()
	assert.Contains(t, v.Content, "Downloading episode 7")
	assert.Contains(t, v.Content, "Ctrl+C")
}

func TestView_DoneSuccessRendersFullBar(t *testing.T) {
	t.Parallel()
	m := &model{
		progress: progress.New(progress.WithDefaultBlend()),
		status:   "Done",
		done:     true,
	}
	v := m.View()
	assert.Contains(t, v.Content, "Done")
}
