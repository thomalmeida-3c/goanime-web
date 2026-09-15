package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HomeItem is an AniList-ranked title. Unlike an earlier version of this
// endpoint, it does NOT try to resolve a playable URL up front — that meant
// dozens of live scrapes per home load (~20s cold) just to grey out the
// titles our sources don't have. Instead the frontend sends the user
// straight to /api/search?q=<title> on click, same as typing it in the
// search bar; that's the one place we already know how to match a title to
// a source, and it only runs for the one anime the user actually picked.
type HomeItem struct {
	AnilistID   int      `json:"anilistId"`
	Title       string   `json:"title"`
	ImageURL    string   `json:"imageUrl"`
	BannerURL   string   `json:"bannerUrl,omitempty"`
	Description string   `json:"description,omitempty"`
	Score       int      `json:"score,omitempty"`
	Genres      []string `json:"genres,omitempty"`
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
		Trending:       toHomeItems(trending),
		SeasonPopular:  toHomeItems(seasonPopular),
		AllTimePopular: toHomeItems(allTimePopular),
		GeneratedAt:    time.Now(),
	}, nil
}

func toHomeItems(media []anilistMedia) []HomeItem {
	items := make([]HomeItem, len(media))
	for i, m := range media {
		title := m.Title.Romaji
		if title == "" {
			title = m.Title.English
		}
		items[i] = HomeItem{
			AnilistID:   m.ID,
			Title:       title,
			ImageURL:    m.CoverImage.Large,
			BannerURL:   m.BannerImage,
			Description: stripHTML(m.Description),
			Score:       m.AverageScore,
			Genres:      m.Genres,
		}
	}
	return items
}
