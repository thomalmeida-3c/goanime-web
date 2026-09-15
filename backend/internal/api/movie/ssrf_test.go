package movie

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDisallowedIP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"loopback", "127.0.0.1", true},
		{"private", "10.0.0.1", true},
		{"multicast", "224.0.0.1", true},
		{"unspecified", "0.0.0.0", true},
		{"public", "8.8.8.8", false},
		{"invalid", "x", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isDisallowedIP(tt.ip))
		})
	}
}

func TestSafeDialFunc_RejectsLoopback(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)
	_, err := safeDialFunc("tcp", srv.Listener.Addr().String(), 2*time.Second, nil)
	assert.Error(t, err)
}

func TestSafeMovieTransport(t *testing.T) {
	t.Parallel()
	tr := safeMovieTransport(7 * time.Second)
	require.NotNil(t, tr)
	assert.NotNil(t, tr.DialContext)
	assert.NotNil(t, tr.DialTLSContext)
	assert.Equal(t, 7*time.Second, tr.TLSHandshakeTimeout)
	assert.Equal(t, 50, tr.MaxIdleConns)
	assert.Equal(t, 10, tr.MaxIdleConnsPerHost)
}
