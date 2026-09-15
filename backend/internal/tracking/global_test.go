package tracking

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Singleton tests share globalTracker — must NOT be parallel.

func resetGlobalTracker(t *testing.T) {
	t.Helper()
	trackerMutex.Lock()
	if globalTracker != nil {
		_ = globalTracker.Close()
	}
	globalTracker = nil
	globalTrackerPath = ""
	trackerMutex.Unlock()
}

func TestGetGlobalTracker_NilWhenUninitialized(t *testing.T) {
	resetGlobalTracker(t)
	t.Cleanup(func() { resetGlobalTracker(t) })

	got := GetGlobalTracker()
	assert.Nil(t, got)
}

func TestGetGlobalTracker_ReturnsCachedAfterNewLocalTracker(t *testing.T) {
	resetGlobalTracker(t)

	// t.TempDir() registers its RemoveAll cleanup first; register the tracker
	// reset (which closes the SQLite handle) afterwards so LIFO order closes
	// the DB before Windows tries to unlink the still-open file.
	dbPath := filepath.Join(t.TempDir(), "tracker.db")
	t.Cleanup(func() { resetGlobalTracker(t) })

	tr := NewLocalTracker(dbPath)
	if tr == nil {
		t.Skip("tracking unavailable (CGO/SQLite not enabled)")
	}

	got := GetGlobalTracker()
	require.NotNil(t, got)
	assert.Same(t, tr, got, "GetGlobalTracker must return the same singleton instance")
}

func TestCloseGlobalTracker_IdempotentWhenNil(t *testing.T) {
	resetGlobalTracker(t)
	t.Cleanup(func() { resetGlobalTracker(t) })

	err := CloseGlobalTracker()
	assert.NoError(t, err)
}

func TestCloseGlobalTracker_ClearsCache(t *testing.T) {
	resetGlobalTracker(t)
	t.Cleanup(func() { resetGlobalTracker(t) })

	dbPath := filepath.Join(t.TempDir(), "tracker.db")
	tr := NewLocalTracker(dbPath)
	if tr == nil {
		t.Skip("tracking unavailable (CGO/SQLite not enabled)")
	}
	require.NotNil(t, GetGlobalTracker())

	require.NoError(t, CloseGlobalTracker())
	assert.Nil(t, GetGlobalTracker())
}

func TestNewLocalTracker_SamePathReturnsCachedSingleton(t *testing.T) {
	resetGlobalTracker(t)

	// Register TempDir cleanup before the tracker reset so the SQLite handle
	// is closed (LIFO) before Windows unlinks the DB file.
	dbPath := filepath.Join(t.TempDir(), "tracker.db")
	t.Cleanup(func() { resetGlobalTracker(t) })

	tr1 := NewLocalTracker(dbPath)
	if tr1 == nil {
		t.Skip("tracking unavailable (CGO/SQLite not enabled)")
	}
	tr2 := NewLocalTracker(dbPath)
	assert.Same(t, tr1, tr2, "Repeat call with same path must reuse the cached tracker")
}
