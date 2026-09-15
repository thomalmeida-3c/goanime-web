package discord

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager_Defaults(t *testing.T) {
	t.Parallel()
	m := NewManager()
	require.NotNil(t, m)
	assert.False(t, m.IsEnabled())
	assert.False(t, m.IsInitialized())
	assert.Equal(t, DiscordClientID, m.GetClientID())
}

func TestManager_SetClientID_BeforeInit(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.SetClientID("custom-id")
	assert.Equal(t, "custom-id", m.GetClientID())
}

func TestManager_SetClientID_BlockedAfterInit(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.isInitialized = true
	m.SetClientID("should-not-apply")
	assert.Equal(t, DiscordClientID, m.GetClientID(), "SetClientID must be a no-op after init")
}

func TestManager_GetInitializationTime_ZeroBeforeInit(t *testing.T) {
	t.Parallel()
	m := NewManager()
	assert.Equal(t, time.Duration(0), m.GetInitializationTime())
}

func TestManager_GetInitializationTime_NonZeroAfterInit(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.isInitialized = true
	m.initTime = time.Now().Add(-2 * time.Second)
	got := m.GetInitializationTime()
	assert.GreaterOrEqual(t, got, 2*time.Second)
	assert.Less(t, got, 5*time.Second)
}

func TestManager_Shutdown_NoOpWhenDisabled(t *testing.T) {
	t.Parallel()
	m := NewManager()
	require.False(t, m.IsEnabled())
	m.Shutdown()
	assert.False(t, m.IsEnabled())
}
