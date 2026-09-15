package player

import (
	"errors"
	"testing"

	"charm.land/bubbles/v2/progress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchProgressAggregatesEpisodeChildren(t *testing.T) {
	parent := &model{}
	ep1 := parent.childProgress("episode-1", 100)
	ep2 := parent.childProgress("episode-2", 200)

	ep1.addProgressReceived(99)
	ep2.addProgressReceived(99)

	parent.mu.Lock()
	received := parent.received
	total := parent.totalBytes
	parent.mu.Unlock()

	assert.Equal(t, int64(198), received)
	assert.Equal(t, int64(300), total)
	assert.InDelta(t, 0.66, float64(received)/float64(total), 0.001)
}

func TestBatchProgressChildTotalAdjustsGlobalTotalWithoutResettingReceived(t *testing.T) {
	parent := &model{}
	ep1 := parent.childProgress("episode-1", 100)
	ep2 := parent.childProgress("episode-2", 200)

	ep1.addProgressReceived(50)
	ep2.addProgressReceived(100)
	ep1.setProgressTotal(120)

	parent.mu.Lock()
	received := parent.received
	total := parent.totalBytes
	parent.mu.Unlock()

	assert.Equal(t, int64(150), received)
	assert.Equal(t, int64(320), total)
}

func TestBatchProgressAbsoluteEpisodeUpdatesDoNotResetGlobalProgress(t *testing.T) {
	parent := &model{}
	ep1 := parent.childProgress("episode-1", 100)
	ep2 := parent.childProgress("episode-2", 100)

	ep1.setProgressReceived(90)
	ep2.setProgressReceived(10)
	ep2.setProgressReceived(80)
	ep1.setProgressReceived(95)

	parent.mu.Lock()
	received := parent.received
	total := parent.totalBytes
	parent.mu.Unlock()

	assert.Equal(t, int64(175), received)
	assert.Equal(t, int64(200), total)
}

func TestBatchProgressFallbackCanResetOnlyCurrentEpisode(t *testing.T) {
	parent := &model{}
	ep1 := parent.childProgress("episode-1", 100)
	ep2 := parent.childProgress("episode-2", 100)

	ep1.addProgressReceived(70)
	ep2.addProgressReceived(40)
	ep2.resetProgressReceived()
	ep2.addProgressReceived(25)

	parent.mu.Lock()
	received := parent.received
	total := parent.totalBytes
	parent.mu.Unlock()

	assert.Equal(t, int64(95), received)
	assert.Equal(t, int64(200), total)
}

func TestBatchProgressFailureStatusDoesNotRenderSuccessMessage(t *testing.T) {
	m := &model{
		progress:   progress.New(progress.WithDefaultBlend()),
		status:     "Downloads completed with 1 failure(s)",
		done:       true,
		err:        errors.New("episode 20: HTTP 404"),
		totalBytes: 100,
		received:   98,
	}

	view := m.View()
	assert.Contains(t, view.Content, "Downloads completed with 1 failure(s)")
	assert.NotContains(t, view.Content, "All downloads completed!")
}

func TestTaskTotal_ReturnsStoredValue(t *testing.T) {
	t.Parallel()
	parent := &model{}
	parent.childProgress("ep-1", 250)
	parent.childProgress("ep-2", 750)

	assert.Equal(t, int64(250), parent.taskTotal("ep-1"))
	assert.Equal(t, int64(750), parent.taskTotal("ep-2"))
	assert.Equal(t, int64(0), parent.taskTotal("missing"))
}

func TestShouldGrowProgressTotal(t *testing.T) {
	t.Parallel()
	parent := &model{}
	parent.setProgressTotal(500)

	tests := []struct {
		name  string
		total int64
		want  bool
	}{
		{"zero rejected", 0, false},
		{"negative rejected", -1, false},
		{"equal does not grow", 500, false},
		{"smaller does not grow", 400, false},
		{"larger grows", 600, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parent.shouldGrowProgressTotal(tt.total))
		})
	}
}

func TestSetProgressPeak_OnlyGrows(t *testing.T) {
	t.Parallel()
	m := &model{}

	m.setProgressPeak(0)
	m.setProgressPeak(-5)
	assert.Equal(t, 0.0, m.peakPct, "non-positive values must be ignored")

	m.setProgressPeak(0.42)
	assert.InDelta(t, 0.42, m.peakPct, 1e-9)

	m.setProgressPeak(0.20)
	assert.InDelta(t, 0.42, m.peakPct, 1e-9, "smaller pct must not lower the peak")

	m.setProgressPeak(0.99)
	assert.InDelta(t, 0.99, m.peakPct, 1e-9)
}

func TestSetProgressPeak_RoutesToParent(t *testing.T) {
	t.Parallel()
	parent := &model{}
	child := parent.childProgress("c", 100)
	require.NotNil(t, child)

	child.setProgressPeak(0.75)
	assert.InDelta(t, 0.75, parent.peakPct, 1e-9)
	assert.Equal(t, 0.0, child.peakPct, "child must delegate peak to parent")
}

func TestSetProgressPeak_NilReceiverIsNoop(t *testing.T) {
	t.Parallel()
	var m *model
	assert.NotPanics(t, func() { m.setProgressPeak(0.5) })
}

func TestChildProgress_ReturnsLinkedChild(t *testing.T) {
	t.Parallel()
	parent := &model{}
	child := parent.childProgress("ep-1", 1000)

	require.NotNil(t, child)
	assert.Equal(t, parent, child.parent)
	assert.Equal(t, "ep-1", child.taskID)
	assert.Equal(t, int64(1000), parent.taskTotal("ep-1"))
	assert.Equal(t, int64(1000), parent.totalBytes)
}

func TestChildProgress_NilParentReturnsNil(t *testing.T) {
	t.Parallel()
	var p *model
	assert.Nil(t, p.childProgress("x", 10))
}

func TestSetTaskTotal_DeltaUpdatesParentTotal(t *testing.T) {
	t.Parallel()
	parent := &model{}
	parent.childProgress("a", 100)
	parent.setTaskTotal("a", 250)
	assert.Equal(t, int64(250), parent.taskTotal("a"))
	assert.Equal(t, int64(250), parent.totalBytes)

	// idempotent: same value → no delta applied
	parent.setTaskTotal("a", 250)
	assert.Equal(t, int64(250), parent.totalBytes)
}

func TestSetProgressTotal_OverwritesParent(t *testing.T) {
	t.Parallel()
	parent := &model{}
	parent.setProgressTotal(500)
	assert.Equal(t, int64(500), parent.totalBytes)

	parent.setProgressTotal(900)
	assert.Equal(t, int64(900), parent.totalBytes)

	parent.setProgressTotal(0)
	assert.Equal(t, int64(900), parent.totalBytes, "zero is rejected")
}

func TestSetProgressTotal_ChildRoutesToParent(t *testing.T) {
	t.Parallel()
	parent := &model{}
	c := parent.childProgress("a", 100)
	c.setProgressTotal(400)
	assert.Equal(t, int64(400), parent.taskTotal("a"))
}

func TestProgressTotal_NilReceiverReturnsZero(t *testing.T) {
	t.Parallel()
	var m *model
	assert.Equal(t, int64(0), m.progressTotal())
}

func TestAddProgressReceived_NilReceiverIsNoop(t *testing.T) {
	t.Parallel()
	var m *model
	assert.NotPanics(t, func() { m.addProgressReceived(10) })
}

func TestAddTaskReceived_AccumulatesIntoParent(t *testing.T) {
	t.Parallel()
	parent := &model{}
	parent.addTaskReceived("a", 30)
	parent.addTaskReceived("a", 20)
	parent.addTaskReceived("b", 50)
	assert.Equal(t, int64(100), parent.received)
}

func TestSetProgressReceived_NegativeRejected(t *testing.T) {
	t.Parallel()
	m := &model{}
	m.setProgressReceived(-5)
	assert.Equal(t, int64(0), m.received)

	m.setProgressReceived(40)
	assert.Equal(t, int64(40), m.received)
}

func TestSetTaskReceived_NeverShrinks(t *testing.T) {
	t.Parallel()
	parent := &model{}
	parent.setTaskReceived("a", 60)
	parent.setTaskReceived("a", 20) // smaller → clamped to prior value
	assert.Equal(t, int64(60), parent.received)
}

func TestResetProgressReceived_NilReceiverIsNoop(t *testing.T) {
	t.Parallel()
	var m *model
	assert.NotPanics(t, func() { m.resetProgressReceived() })
}

func TestResetTaskReceived_RemovesEntryAndAdjustsTotal(t *testing.T) {
	t.Parallel()
	parent := &model{}
	parent.addTaskReceived("a", 80)
	parent.addTaskReceived("b", 20)
	parent.resetTaskReceived("a")
	assert.Equal(t, int64(20), parent.received)
}

func TestResetTaskReceived_UnknownTaskNoop(t *testing.T) {
	t.Parallel()
	parent := &model{}
	parent.addTaskReceived("a", 10)
	parent.resetTaskReceived("nonexistent")
	assert.Equal(t, int64(10), parent.received)
}
