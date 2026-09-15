package main

import (
	"context"
	"fmt"
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
	mux.HandleFunc("GET /api/manga/chapters", handleMangaChapters)
	mux.HandleFunc("GET /api/manga/pages", handleChapterPages)
}

// handleMangaHome serves the "popular" manga row, cached like /api/home —
// same reasoning: cheap to cache, no reason to hit MangaDex on every load.
// MangaLivre doesn't expose anything like a follower-count ranking, so this
// row stays MangaDex-only; MangaLivre's value is in search/reading, where
// its full chapter runs matter more than home-page ranking.
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

// handleMangaSearch fans out to both sources in parallel and merges the
// results, then re-ranks the combined list by relevance to the query so the
// best title match wins regardless of which source found it — otherwise a
// MangaDex spinoff and a MangaLivre main series would just land in
// source-then-source order instead of best-match-first.
func handleMangaSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, errQueryRequired)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var (
		wg                   sync.WaitGroup
		dexItems, livreItems []MangaItem
		dexErr, livreErr     error
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		dexItems, dexErr = searchMangaDex(ctx, q, 24)
	}()
	go func() {
		defer wg.Done()
		livreItems, livreErr = searchMangaLivre(ctx, q, 24)
	}()
	wg.Wait()

	items := append(dexItems, livreItems...)
	if len(items) == 0 && (dexErr != nil || livreErr != nil) {
		writeError(w, http.StatusBadGateway, fmt.Errorf("mangadex: %v, mangalivre: %v", dexErr, livreErr))
		return
	}

	rankMangaByRelevance(q, items)
	writeJSON(w, http.StatusOK, items)
}

// handleMangaChapters dispatches to the right source by the `source` query
// param — MangaDex and MangaLivre have unrelated id schemes (a MangaDex
// manga is a UUID, a MangaLivre one is its page URL), so the request has to
// say which backend `id` belongs to.
func handleMangaChapters(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	source := r.URL.Query().Get("source")
	if id == "" {
		writeError(w, http.StatusBadRequest, errMangaIDRequired)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var (
		chapters []ChapterItem
		err      error
	)
	switch source {
	case "MangaLivre":
		chapters, err = fetchMangaLivreChapters(ctx, id)
	default:
		var raw []mdChapter
		raw, err = fetchMangaChapters(ctx, id)
		if err == nil {
			chapters = toChapterItems(raw)
		}
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, chapters)
}

// handleChapterPages resolves this read's page image URLs. MangaDex needs a
// fresh at-home/server CDN handshake per read; MangaLivre's are just static
// files on their own domain, read straight off the chapter page. Either
// way, the frontend loads the images directly — no proxy needed, unlike the
// anime video sources.
func handleChapterPages(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	source := r.URL.Query().Get("source")
	if id == "" {
		writeError(w, http.StatusBadRequest, errChapterIDRequired)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var (
		pages []string
		err   error
	)
	switch source {
	case "MangaLivre":
		pages, err = resolveMangaLivreChapterPages(ctx, id)
	default:
		pages, err = resolveChapterPages(ctx, id)
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"pages": pages})
}
