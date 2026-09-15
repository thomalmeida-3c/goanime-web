package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HomeItem is an AniList-ranked title, optionally linked to a playable
// Goyabu anime when findGoyabuMatch found a confident match. AnimeURL/Source
// are empty when no match was found — the frontend should show the card
// without a play affordance in that case.
type HomeItem struct {
	AnilistID   int      `json:"anilistId"`
	Title       string   `json:"title"`
	ImageURL    string   `json:"imageUrl"`
	BannerURL   string   `json:"bannerUrl,omitempty"`
	Description string   `json:"description,omitempty"`
	Score       int      `json:"score,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	AnimeURL    string   `json:"animeUrl,omitempty"`
	Source      string   `json:"source,omitempty"`
}

type HomeResponse struct {
	Trending       []HomeItem `json:"trending"`
	SeasonPopular  []HomeItem `json:"seasonPopular"`
	AllTimePopular []HomeItem `json:"allTimePopular"`
	GeneratedAt    time.Time  `json:"generatedAt"`
}

const (
	homeCacheTTL    = time.Hour
	homeItemsPerRow = 12
)

var (
	homeCacheMu sync.Mutex
	homeCache   *HomeResponse
	homeBuildMu sync.Mutex // serializes rebuilds so concurrent cold requests don't all scrape at once
)

// getHome returns the cached home payload, rebuilding it first if it's
// missing or stale. Building is expensive (dozens of live Goyabu searches),
// so it's cached for homeCacheTTL and rebuilds are serialized.
func getHome(ctx context.Context) (*HomeResponse, error) {
	homeCacheMu.Lock()
	fresh := homeCache != nil && time.Since(homeCache.GeneratedAt) < homeCacheTTL
	cached := homeCache
	homeCacheMu.Unlock()

	if fresh {
		return cached, nil
	}

	homeBuildMu.Lock()
	defer homeBuildMu.Unlock()

	// Another goroutine may have rebuilt it while we waited for the lock.
	homeCacheMu.Lock()
	fresh = homeCache != nil && time.Since(homeCache.GeneratedAt) < homeCacheTTL
	cached = homeCache
	homeCacheMu.Unlock()
	if fresh {
		return cached, nil
	}

	built, err := buildHome(ctx)
	if err != nil {
		return nil, err
	}

	homeCacheMu.Lock()
	homeCache = built
	homeCacheMu.Unlock()

	return built, nil
}

func buildHome(ctx context.Context) (*HomeResponse, error) {
	season, year := currentAnimeSeason(time.Now())

	trending, err := fetchAnilist(ctx, anilistFetchParams{
		Sort: []string{"TRENDING_DESC"}, PerPage: homeItemsPerRow,
	})
	if err != nil {
		return nil, fmt.Errorf("anilist trending: %w", err)
	}

	seasonPopular, err := fetchAnilist(ctx, anilistFetchParams{
		Sort: []string{"POPULARITY_DESC"}, Season: season, SeasonYear: year, PerPage: homeItemsPerRow,
	})
	if err != nil {
		return nil, fmt.Errorf("anilist season popular: %w", err)
	}

	allTimePopular, err := fetchAnilist(ctx, anilistFetchParams{
		Sort: []string{"POPULARITY_DESC"}, PerPage: homeItemsPerRow,
	})
	if err != nil {
		return nil, fmt.Errorf("anilist all-time popular: %w", err)
	}

	return &HomeResponse{
		Trending:       matchAll(trending),
		SeasonPopular:  matchAll(seasonPopular),
		AllTimePopular: matchAll(allTimePopular),
		GeneratedAt:    time.Now(),
	}, nil
}
