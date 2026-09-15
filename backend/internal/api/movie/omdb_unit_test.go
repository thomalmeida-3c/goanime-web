package movie

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestOMDbClient(t *testing.T, handler http.HandlerFunc) *OMDbClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &OMDbClient{
		client:  srv.Client(),
		apiKey:  "test-key",
		baseURL: srv.URL,
	}
}

func TestOMDbClient_IsConfigured(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{"with key", "abc123", true},
		{"empty key", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &OMDbClient{apiKey: tt.apiKey}
			assert.Equal(t, tt.want, c.IsConfigured())
		})
	}
}

func TestOMDbClient_NewOMDbClient_DefaultsToTrilogyKey(t *testing.T) {
	// Cannot run in parallel because mutates env
	t.Setenv("OMDB_API_KEY", "")
	c := NewOMDbClient()
	require.NotNil(t, c)
	assert.Equal(t, "trilogy", c.apiKey)
	assert.Equal(t, OMDbBaseURL, c.baseURL)
}

func TestOMDbClient_NewOMDbClient_UsesEnvKey(t *testing.T) {
	t.Setenv("OMDB_API_KEY", "real-key")
	c := NewOMDbClient()
	assert.Equal(t, "real-key", c.apiKey)
}

func TestOMDbClient_makeRequest_Success(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ok":true}`)
	})
	body, err := c.makeRequest(c.baseURL + "/?apikey=test-key")
	require.NoError(t, err)
	assert.JSONEq(t, `{"ok":true}`, string(body))
}

func TestOMDbClient_makeRequest_Non2xx(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	body, err := c.makeRequest(c.baseURL + "/")
	require.Error(t, err)
	assert.Nil(t, body)
	assert.Contains(t, err.Error(), "OMDb API returned status")
}

func TestOMDbClient_makeRequest_BadURL(t *testing.T) {
	t.Parallel()
	c := &OMDbClient{client: &http.Client{}}
	_, err := c.makeRequest("://invalid")
	require.Error(t, err)
}

func TestOMDbClient_SearchByTitle_Mock(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.URL.Query().Get("apikey"))
		assert.Equal(t, "Inception", r.URL.Query().Get("s"))
		assert.Equal(t, "movie", r.URL.Query().Get("type"))
		fmt.Fprint(w, `{"Search":[{"Title":"Inception","Year":"2010","imdbID":"tt1375666","Type":"movie"}],"totalResults":"1","Response":"True"}`)
	})
	got, err := c.SearchByTitle("Inception", "movie")
	require.NoError(t, err)
	require.Len(t, got.Search, 1)
	assert.Equal(t, "Inception", got.Search[0].Title)
	assert.Equal(t, "tt1375666", got.Search[0].IMDBID)
}

func TestOMDbClient_SearchByTitle_NoTypeFilter(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("type"))
		fmt.Fprint(w, `{"Response":"True","Search":[]}`)
	})
	_, err := c.SearchByTitle("Foo", "")
	require.NoError(t, err)
}

func TestOMDbClient_SearchByTitle_ResponseFalse(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Response":"False","Error":"Movie not found!"}`)
	})
	_, err := c.SearchByTitle("Whatever", "movie")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Movie not found!")
}

func TestOMDbClient_SearchByTitle_BadJSON(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html>not json</html>`)
	})
	_, err := c.SearchByTitle("Foo", "movie")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestOMDbClient_GetByIMDBID_Mock(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "tt1375666", r.URL.Query().Get("i"))
		assert.Equal(t, "short", r.URL.Query().Get("plot"))
		fmt.Fprint(w, `{"Title":"Inception","Runtime":"148 min","imdbRating":"8.8","Genre":"Action, Sci-Fi","Response":"True"}`)
	})
	got, err := c.GetByIMDBID("tt1375666")
	require.NoError(t, err)
	assert.Equal(t, "Inception", got.Title)
	assert.Equal(t, 148, got.GetRuntimeMinutes())
}

func TestOMDbClient_GetByIMDBID_ResponseFalse(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Response":"False","Error":"Incorrect IMDb ID."}`)
	})
	_, err := c.GetByIMDBID("ttxx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Incorrect IMDb ID.")
}

func TestOMDbClient_GetByIMDBID_BadJSON(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `not json`)
	})
	_, err := c.GetByIMDBID("tt0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestOMDbClient_GetByTitle_WithYear(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Inception", r.URL.Query().Get("t"))
		assert.Equal(t, "2010", r.URL.Query().Get("y"))
		fmt.Fprint(w, `{"Title":"Inception","Year":"2010","imdbID":"tt1375666","Response":"True"}`)
	})
	got, err := c.GetByTitle("Inception", "2010")
	require.NoError(t, err)
	assert.Equal(t, "tt1375666", got.IMDBID)
}

func TestOMDbClient_GetByTitle_WithoutYear(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("y"))
		fmt.Fprint(w, `{"Response":"True","Title":"X"}`)
	})
	_, err := c.GetByTitle("X", "")
	require.NoError(t, err)
}

func TestOMDbClient_GetByTitle_ResponseFalse(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Response":"False","Error":"Movie not found!"}`)
	})
	_, err := c.GetByTitle("Foo", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Movie not found!")
}

func TestOMDbClient_GetByTitle_BadJSON(t *testing.T) {
	t.Parallel()
	c := newTestOMDbClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<<<`)
	})
	_, err := c.GetByTitle("Foo", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}
