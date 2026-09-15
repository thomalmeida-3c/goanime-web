package api

import (
	"context"
	"net"
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
		{"loopback v4", "127.0.0.1", true},
		{"loopback v6", "::1", true},
		{"private 10", "10.0.0.1", true},
		{"private 192.168", "192.168.1.1", true},
		{"private 172.16", "172.16.0.1", true},
		{"multicast", "224.0.0.1", true},
		{"unspecified", "0.0.0.0", true},
		{"public 8.8.8.8", "8.8.8.8", false},
		{"public ipv6", "2001:4860:4860::8888", false},
		{"invalid", "not-an-ip", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsDisallowedIP(tt.ip))
		})
	}
}

// checkDisallowedIP closes the conn when ip is disallowed. Validate with a
// pipe-backed conn whose RemoteAddr resolves to 127.0.0.1.
type testConn struct {
	net.Conn
	remote net.Addr
	closed bool
}

func (c *testConn) RemoteAddr() net.Addr { return c.remote }
func (c *testConn) Close() error         { c.closed = true; return nil }

type testAddr struct{ s string }

func (a testAddr) Network() string { return "tcp" }
func (a testAddr) String() string  { return a.s }

func TestCheckDisallowedIP(t *testing.T) {
	t.Parallel()

	t.Run("loopback rejected", func(t *testing.T) {
		t.Parallel()
		c := &testConn{remote: testAddr{"127.0.0.1:80"}}
		err := checkDisallowedIP(c)
		assert.Error(t, err)
		assert.True(t, c.closed)
	})

	t.Run("public allowed", func(t *testing.T) {
		t.Parallel()
		c := &testConn{remote: testAddr{"8.8.8.8:80"}}
		err := checkDisallowedIP(c)
		assert.NoError(t, err)
		assert.False(t, c.closed)
	})

	t.Run("bad address closes", func(t *testing.T) {
		t.Parallel()
		c := &testConn{remote: testAddr{"not-a-host-port"}}
		err := checkDisallowedIP(c)
		assert.Error(t, err)
		assert.True(t, c.closed)
	})
}

func TestDialFunc_RejectsLoopback(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)

	host := srv.Listener.Addr().String()
	_, err := dialFunc("tcp", host, 2*time.Second, nil)
	assert.Error(t, err)
}

func TestSafeTransport_Configured(t *testing.T) {
	t.Parallel()
	tr := SafeTransport(5 * time.Second)
	require.NotNil(t, tr)
	assert.NotNil(t, tr.DialContext)
	assert.NotNil(t, tr.DialTLSContext)
	assert.Equal(t, 5*time.Second, tr.TLSHandshakeTimeout)
}

func TestSafeGet_RejectsLoopback(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	_, err := SafeGet(srv.URL)
	assert.Error(t, err)
}

func TestValidateExternalURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"invalid scheme parses ok no host", "::nope", true},
		{"loopback ip literal", "http://127.0.0.1/x", true},
		{"private ip literal", "http://10.0.0.1/x", true},
		{"public ip literal", "http://8.8.8.8/x", false},
		{"no hostname", "http://", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateExternalURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestSafeDialContext_RejectsLoopback(t *testing.T) {
	t.Parallel()
	dial := SafeDialContext(2 * time.Second)
	require.NotNil(t, dial)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)

	_, err := dial(context.Background(), "tcp", srv.Listener.Addr().String())
	assert.Error(t, err)
}
