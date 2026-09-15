package movie

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockTMDBClient(t *testing.T, handler http.HandlerFunc) *TMDBClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &TMDBClient{
		client:    srv.Client(),
		apiKey:    "test-key",
		baseURL:   srv.URL,
		imageBase: TMDBImageBaseURL,
	}
}

func TestTMDBClient_IsConfigured(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{"with key", "abc", true},
		{"empty key", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &TMDBClient{apiKey: tt.apiKey}
			assert.Equal(t, tt.want, c.IsConfigured())
		})
	}
}

func TestTMDBClient_NewTMDBClient_UsesEnv(t *testing.T) {
	t.Setenv("TMDB_API_KEY", "x")
	c := NewTMDBClient()
	require.NotNil(t, c)
	assert.Equal(t, "x", c.apiKey)
	assert.Equal(t, TMDBBaseURL, c.baseURL)
}

func TestTMDBClient_NewTMDBClient_EmptyEnv(t *testing.T) {
	t.Setenv("TMDB_API_KEY", "")
	c := NewTMDBClient()
	assert.Empty(t, c.apiKey)
	assert.False(t, c.IsConfigured())
}

func TestTMDBClient_makeRequest_AppendsKey(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.URL.Query().Get("api_key"))
		assert.Equal(t, "en-US", r.URL.Query().Get("language"))
		fmt.Fprint(w, `{"ok":true}`)
	})
	body, err := c.makeRequest(c.baseURL + "/movie/1?language=en-US")
	require.NoError(t, err)
	assert.JSONEq(t, `{"ok":true}`, string(body))
}

func TestTMDBClient_makeRequest_NoQueryString(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.URL.Query().Get("api_key"))
		fmt.Fprint(w, `{}`)
	})
	_, err := c.makeRequest(c.baseURL + "/something")
	require.NoError(t, err)
}

func TestTMDBClient_makeRequest_Non2xx(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	_, err := c.makeRequest(c.baseURL + "/movie/1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TMDB API returned status")
}

func TestTMDBClient_makeRequest_BadURL(t *testing.T) {
	t.Parallel()
	c := &TMDBClient{client: &http.Client{}, apiKey: "k"}
	_, err := c.makeRequest("://broken")
	require.Error(t, err)
}

func TestTMDBClient_SearchMulti_FiltersToMovieAndTV(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/search/multi")
		assert.Equal(t, "Foo", r.URL.Query().Get("query"))
		fmt.Fprint(w, `{"results":[
			{"id":1,"title":"A","media_type":"movie"},
			{"id":2,"name":"B","media_type":"tv"},
			{"id":3,"name":"P","media_type":"person"}
		]}`)
	})
	got, err := c.SearchMulti("Foo")
	require.NoError(t, err)
	require.Len(t, got.Results, 2)
	assert.Equal(t, "movie", got.Results[0].MediaType)
	assert.Equal(t, "tv", got.Results[1].MediaType)
}

func TestTMDBClient_SearchMulti_BadJSON(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `not json`)
	})
	_, err := c.SearchMulti("x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestTMDBClient_GetTVSeasons(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/tv/42")
		fmt.Fprint(w, `{"seasons":[{"season_number":1,"episode_count":12},{"season_number":2,"episode_count":13}]}`)
	})
	seasons, err := c.GetTVSeasons(42)
	require.NoError(t, err)
	require.Len(t, seasons, 2)
	assert.Equal(t, 1, seasons[0].SeasonNumber)
	assert.Equal(t, 12, seasons[0].EpisodeCount)
}

func TestTMDBClient_GetTVSeasons_BadJSON(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `xxx`)
	})
	_, err := c.GetTVSeasons(1)
	require.Error(t, err)
}

func TestTMDBClient_GetTVSeasons_HTTPError(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := c.GetTVSeasons(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get TV seasons")
}

func TestTMDBClient_GetSeasonEpisodes(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/tv/42/season/3")
		fmt.Fprint(w, `{"episodes":[{"episode_number":1,"name":"Pilot"},{"episode_number":2,"name":"Two"}]}`)
	})
	eps, err := c.GetSeasonEpisodes(42, 3)
	require.NoError(t, err)
	require.Len(t, eps, 2)
	assert.Equal(t, "Pilot", eps[0].Name)
}

func TestTMDBClient_GetSeasonEpisodes_BadJSON(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<bad>`)
	})
	_, err := c.GetSeasonEpisodes(1, 1)
	require.Error(t, err)
}

func TestTMDBClient_GetSeasonEpisodes_HTTPError(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	_, err := c.GetSeasonEpisodes(1, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get season episodes")
}

func TestTMDBClient_GetCredits(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/movie/7/credits")
		fmt.Fprint(w, `{"cast":[{"name":"Actor One"}],"crew":[{"name":"Dir One","job":"Director"}]}`)
	})
	got, err := c.GetCredits("movie", 7)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, got.Cast, 1)
	assert.Equal(t, "Actor One", got.Cast[0].Name)
}

func TestTMDBClient_GetCredits_BadJSON(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `nope`)
	})
	_, err := c.GetCredits("tv", 1)
	require.Error(t, err)
}

func TestTMDBClient_GetCredits_HTTPError(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	_, err := c.GetCredits("tv", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get credits")
}

func TestTMDBClient_FindByIMDBID_MovieMatch(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/find/tt1234567")
		assert.Equal(t, "imdb_id", r.URL.Query().Get("external_source"))
		fmt.Fprint(w, `{"movie_results":[{"id":1,"title":"M"}],"tv_results":[]}`)
	})
	got, err := c.FindByIMDBID("tt1234567")
	require.NoError(t, err)
	assert.Equal(t, "movie", got.MediaType)
	assert.Equal(t, 1, got.ID)
}

func TestTMDBClient_FindByIMDBID_TVFallback(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"movie_results":[],"tv_results":[{"id":2,"name":"T"}]}`)
	})
	got, err := c.FindByIMDBID("tt0000002")
	require.NoError(t, err)
	assert.Equal(t, "tv", got.MediaType)
	assert.Equal(t, 2, got.ID)
}

func TestTMDBClient_FindByIMDBID_NoMatch(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"movie_results":[],"tv_results":[]}`)
	})
	_, err := c.FindByIMDBID("tt000")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no results found")
}

func TestTMDBClient_FindByIMDBID_BadJSON(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `garbage`)
	})
	_, err := c.FindByIMDBID("tt0")
	require.Error(t, err)
}

func TestTMDBClient_FindByIMDBID_HTTPError(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := c.FindByIMDBID("tt0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find by IMDB ID")
}

func TestTMDBClient_GetTrending_DefaultsToAllWeek(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/trending/all/week")
		fmt.Fprint(w, `{"results":[]}`)
	})
	_, err := c.GetTrending("", "")
	require.NoError(t, err)
}

func TestTMDBClient_GetTrending_CustomParams(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/trending/movie/day")
		fmt.Fprint(w, `{"results":[{"id":1}]}`)
	})
	got, err := c.GetTrending("movie", "day")
	require.NoError(t, err)
	require.Len(t, got.Results, 1)
}

func TestTMDBClient_GetTrending_BadJSON(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<>`)
	})
	_, err := c.GetTrending("movie", "day")
	require.Error(t, err)
}

func TestTMDBClient_GetTrending_HTTPError(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	_, err := c.GetTrending("", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get trending")
}

func TestTMDBClient_GetPopular(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/movie/popular")
		fmt.Fprint(w, `{"results":[{"id":1,"title":"A"},{"id":2,"title":"B"}]}`)
	})
	got, err := c.GetPopular("movie")
	require.NoError(t, err)
	require.Len(t, got.Results, 2)
	assert.Equal(t, "movie", got.Results[0].MediaType)
	assert.Equal(t, "movie", got.Results[1].MediaType)
}

func TestTMDBClient_GetPopular_BadJSON(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `bad`)
	})
	_, err := c.GetPopular("tv")
	require.Error(t, err)
}

func TestTMDBClient_GetPopular_HTTPError(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	_, err := c.GetPopular("tv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get popular")
}

func TestTMDBClient_SearchMovies_BadJSON(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<>`)
	})
	_, err := c.SearchMovies("x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestTMDBClient_SearchMovies_MarksMediaType(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"results":[{"id":1,"title":"A"},{"id":2,"title":"B"}]}`)
	})
	got, err := c.SearchMovies("x")
	require.NoError(t, err)
	for _, r := range got.Results {
		assert.Equal(t, "movie", r.MediaType)
	}
}

func TestTMDBClient_SearchTV_BadJSON(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<>`)
	})
	_, err := c.SearchTV("x")
	require.Error(t, err)
}

func TestTMDBClient_SearchTV_MarksMediaType(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"results":[{"id":1,"name":"A"}]}`)
	})
	got, err := c.SearchTV("x")
	require.NoError(t, err)
	assert.Equal(t, "tv", got.Results[0].MediaType)
}

func TestTMDBClient_GetMovieDetails_BadJSON(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `bad`)
	})
	_, err := c.GetMovieDetails(1)
	require.Error(t, err)
}

func TestTMDBClient_GetMovieDetails_HTTPError(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := c.GetMovieDetails(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get movie details")
}

func TestTMDBClient_GetTVDetails(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/tv/55")
		fmt.Fprint(w, `{"id":55,"name":"Show"}`)
	})
	got, err := c.GetTVDetails(55)
	require.NoError(t, err)
	assert.Equal(t, "Show", got.Name)
}

func TestTMDBClient_GetTVDetails_BadJSON(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `nope`)
	})
	_, err := c.GetTVDetails(1)
	require.Error(t, err)
}

func TestTMDBClient_GetTVDetails_HTTPError(t *testing.T) {
	t.Parallel()
	c := newMockTMDBClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	_, err := c.GetTVDetails(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get TV details")
}

func TestTMDBClient_GetImageURL_DefaultSize(t *testing.T) {
	t.Parallel()
	c := &TMDBClient{imageBase: TMDBImageBaseURL}
	got := c.GetImageURL("/a.jpg", "")
	assert.Equal(t, "https://image.tmdb.org/t/p/w500/a.jpg", got)
}
