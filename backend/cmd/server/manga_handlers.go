package main

import (
	"context"
	"net/http"
	"sync"
	"time"
)

const mangaHomeCacheTTL = time.Hour

var (
	mangaHomeCacheMu sync.Mutex
	mangaHomeCache   []MangaItem
	mangaHomeCacheAt time.Time
)

func registerMangaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/manga/home", handleMangaHome)
	mux.HandleFunc("GET /api/manga/search", handleMangaSearch)
	mux.HandleFunc("GET /api/manga/{id}/chapters", handleMangaChapters)
	mux.HandleFunc("GET /api/manga/chapter/{id}/pages", handleChapterPages)
}

// handleMangaHome serves the "popular" manga row, cached like /api/home —
// same reasoning: cheap to cache, no reason to hit MangaDex on every load.
func handleMangaHome(w http.ResponseWriter, r *http.Request) {
	mangaHomeCacheMu.Lock()
	if mangaHomeCache != nil && time.Since(mangaHomeCacheAt) < mangaHomeCacheTTL {
		cached := mangaHomeCache
		mangaHomeCacheMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"popular": cached})
		return
	}
	mangaHomeCacheMu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	items, err := fetchMangaDexPopular(ctx, 18)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	mangaHomeCacheMu.Lock()
	mangaHomeCache = items
	mangaHomeCacheAt = time.Now()
	mangaHomeCacheMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{"popular": items})
}

func handleMangaSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, errQueryRequired)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	items, err := searchMangaDex(ctx, q, 24)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func handleMangaChapters(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	chapters, err := fetchMangaChapters(ctx, id)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, toChapterItems(chapters))
}

// handleChapterPages resolves a fresh MangaDex@Home CDN URL for this read —
// the frontend loads the images directly from that CDN (they're served
// specifically for direct browser <img> loading, no proxy needed, unlike
// the anime video sources).
func handleChapterPages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	pages, err := resolveChapterPages(ctx, id)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"pages": pages})
}
