package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/alvarorichard/Goanime/internal/player"
	"github.com/alvarorichard/Goanime/pkg/goanime"
	"github.com/alvarorichard/Goanime/pkg/goanime/types"
)

var (
	client = goanime.NewClient()
	store  = newSessionStore()
)

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/search", handleSearch)
	mux.HandleFunc("GET /api/episodes", handleEpisodes)
	mux.HandleFunc("GET /api/stream", handleStream)
	mux.HandleFunc("GET /api/proxy/{id}", handleProxy)
	mux.HandleFunc("GET /api/health", handleHealth)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// handleSearch fans out across every registered scraper source (AnimeFire,
// Goyabu, SuperFlix, AniDB) and drops any result whose source string we
// can't map back to a Source (types.ParseSource) — that would otherwise let
// the frontend list an anime that 400s the moment you click it.
func handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, errQueryRequired)
		return
	}

	results, err := client.SearchAnime(q, nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	usable := make([]*types.Anime, 0, len(results))
	for _, anime := range results {
		if _, err := types.ParseSource(anime.Source); err == nil {
			usable = append(usable, anime)
		}
	}

	writeJSON(w, http.StatusOK, usable)
}

func handleEpisodes(w http.ResponseWriter, r *http.Request) {
	animeURL := r.URL.Query().Get("url")
	sourceStr := r.URL.Query().Get("source")
	if animeURL == "" || sourceStr == "" {
		writeError(w, http.StatusBadRequest, errURLAndSourceRequired)
		return
	}

	source, err := types.ParseSource(sourceStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	episodes, err := client.GetAnimeEpisodes(animeURL, source)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, episodes)
}

// handleStream resolves the episode's real stream URL + required headers via
// the GoAnime SDK, then stashes them server-side and hands the frontend a
// same-origin playbackUrl that proxies through handleProxy — the browser
// itself never needs to know the upstream URL or send provider-specific
// headers.
func handleStream(w http.ResponseWriter, r *http.Request) {
	episodeURL := r.URL.Query().Get("episodeUrl")
	sourceStr := r.URL.Query().Get("source")
	if episodeURL == "" || sourceStr == "" {
		writeError(w, http.StatusBadRequest, errEpisodeAndSourceRequired)
		return
	}

	if _, err := types.ParseSource(sourceStr); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	opts := goanime.DefaultStreamOptions()
	if q := r.URL.Query().Get("quality"); q != "" {
		opts.Quality = q
	}
	if m := r.URL.Query().Get("mode"); m != "" {
		opts.Mode = m
	}

	anime := &types.Anime{Source: sourceStr}
	episode := &types.Episode{URL: episodeURL}

	streamURL, metadata, err := client.GetEpisodeStreamURL(anime, episode, &opts)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	// Goyabu (and other Blogger-hosted sources) hand back a blogger.com/video.g
	// page, not a directly playable URL — it needs the batchexecute resolution
	// the CLI does before handing mpv a URL. Do the same here: swap it for a
	// local proxy URL that already streams the resolved googlevideo CDN response.
	if strings.HasPrefix(streamURL, "https://www.blogger.com/video.g") {
		resolved, err := player.ResolveBloggerStream(streamURL)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		streamURL = resolved
		metadata = nil
	}

	id := store.put(streamSession{
		URL:     streamURL,
		Headers: metadata,
		Created: time.Now(),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"id":          id,
		"playbackUrl": "/api/proxy/" + id,
		"streamUrl":   streamURL,
		"metadata":    metadata,
	})
}
