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

	mangaLatestCacheMu sync.Mutex
	mangaLatestCache   []MangaItem
	mangaLatestCacheAt time.Time
)

func registerMangaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/manga/home", handleMangaHome)
	mux.HandleFunc("GET /api/manga/latest", handleMangaLatest)
	mux.HandleFunc("GET /api/manga/search", handleMangaSearch)
	mux.HandleFunc("GET /api/manga/chapters", handleMangaChapters)
	mux.HandleFunc("GET /api/manga/pages", handleChapterPages)
}

// fanMangaLivre runs the mangalivre.to and mangalivre.blog fetchers for one
// listing in parallel and merges what comes back — same idea as /api/search
// fanning anime sources out. Neither failing the other: an error from one
// source just means fewer results, not a broken request (mirrors how the
// anime search already tolerates a source being down).
func fanMangaLivre(
	toFn func(context.Context, int) ([]MangaItem, error),
	blogFn func(context.Context, int) ([]MangaItem, error),
	ctx context.Context,
	limit int,
) []MangaItem {
	var wg sync.WaitGroup
	var to, blog []MangaItem
	wg.Add(2)
	go func() {
		defer wg.Done()
		to, _ = toFn(ctx, limit)
	}()
	go func() {
		defer wg.Done()
		blog, _ = blogFn(ctx, limit)
	}()
	wg.Wait()

	merged := make([]MangaItem, 0, len(to)+len(blog))
	merged = append(merged, to...)
	merged = append(merged, blog...)
	return merged
}

// handleMangaLatest serves the "recém adicionados" row for the dedicated
// Mangás page, cached the same way as handleMangaHome.
func handleMangaLatest(w http.ResponseWriter, r *http.Request) {
	mangaLatestCacheMu.Lock()
	if mangaLatestCache != nil && time.Since(mangaLatestCacheAt) < mangaHomeCacheTTL {
		cached := mangaLatestCache
		mangaLatestCacheMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"latest": cached})
		return
	}
	mangaLatestCacheMu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	items := fanMangaLivre(fetchMangaLivreLatest, fetchMangaLivreBlogLatest, ctx, 12)

	mangaLatestCacheMu.Lock()
	mangaLatestCache = items
	mangaLatestCacheAt = time.Now()
	mangaLatestCacheMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{"latest": items})
}

// handleMangaHome serves the "popular" manga row, cached like /api/home —
// same reasoning: cheap to cache, no reason to hit the source on every
// load. MangaDex support (mangadex.go) is still here and working, just not
// wired into the listing right now — that one stayed out deliberately (see
// README "MangaDex"); mangalivre.to and mangalivre.blog are both real,
// working sources with different catalogs, so both are fanned out here.
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

	items := fanMangaLivre(fetchMangaLivrePopular, fetchMangaLivreBlogPopular, ctx, 12)

	mangaHomeCacheMu.Lock()
	mangaHomeCache = items
	mangaHomeCacheAt = time.Now()
	mangaHomeCacheMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{"popular": items})
}

// handleMangaSearch fans out to both MangaLivre sources in parallel and
// merges — see fanMangaLivre. MangaDex is still there, just not called from
// the listing right now (see README "MangaDex").
func handleMangaSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, errQueryRequired)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	items := fanMangaLivre(
		func(ctx context.Context, limit int) ([]MangaItem, error) { return searchMangaLivre(ctx, q, limit) },
		func(ctx context.Context, limit int) ([]MangaItem, error) { return searchMangaLivreBlog(ctx, q, limit) },
		ctx, 24,
	)

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
	case "MangaLivre.blog":
		chapters, err = fetchMangaLivreBlogChapters(ctx, id)
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
	case "MangaLivre.blog":
		pages, err = resolveMangaLivreBlogChapterPages(ctx, id)
	default:
		pages, err = resolveChapterPages(ctx, id)
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"pages": pages})
}
