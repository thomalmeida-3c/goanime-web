package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const anilistEndpoint = "https://graphql.anilist.co"

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	return strings.TrimSpace(htmlTagRe.ReplaceAllString(s, " "))
}

// anilistMedia is the subset of AniList's Media object we need for home-page
// cards. AniList is used purely for real, global popularity rankings — our
// own scrapers (Goyabu/SuperFlix/...) have no view-count data of their own.
type anilistMedia struct {
	ID    int `json:"id"`
	Title struct {
		Romaji  string `json:"romaji"`
		English string `json:"english"`
	} `json:"title"`
	CoverImage struct {
		Large string `json:"large"`
	} `json:"coverImage"`
	BannerImage  string   `json:"bannerImage"`
	Description  string   `json:"description"`
	AverageScore int      `json:"averageScore"`
	Genres       []string `json:"genres"`
}

type anilistPageResponse struct {
	Data struct {
		Page struct {
			Media []anilistMedia `json:"media"`
		} `json:"Page"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

const anilistQuery = `
query ($sort: [MediaSort], $season: MediaSeason, $seasonYear: Int, $perPage: Int) {
  Page(page: 1, perPage: $perPage) {
    media(type: ANIME, sort: $sort, season: $season, seasonYear: $seasonYear, isAdult: false) {
      id
      title { romaji english }
      coverImage { large }
      bannerImage
      description(asHtml: false)
      averageScore
      genres
    }
  }
}
`

type anilistFetchParams struct {
	Sort       []string
	Season     string // "" to omit (all-time / trending queries don't filter by season)
	SeasonYear int    // 0 to omit
	PerPage    int
}

// fetchAnilist queries AniList's public GraphQL API (no key required).
func fetchAnilist(ctx context.Context, params anilistFetchParams) ([]anilistMedia, error) {
	variables := map[string]any{
		"sort":    params.Sort,
		"perPage": params.PerPage,
	}
	if params.Season != "" {
		variables["season"] = params.Season
	}
	if params.SeasonYear != 0 {
		variables["seasonYear"] = params.SeasonYear
	}

	payload, err := json.Marshal(map[string]any{
		"query":     anilistQuery,
		"variables": variables,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed anilistPageResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding AniList response: %w", err)
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("AniList error: %s", parsed.Errors[0].Message)
	}
	return parsed.Data.Page.Media, nil
}

// currentAnimeSeason maps a date to the AniList season/year it falls in
// (Dec/Jan/Feb -> Winter, etc.), following AniList's own convention where
// December belongs to the following year's Winter season.
func currentAnimeSeason(t time.Time) (season string, year int) {
	month := int(t.Month())
	year = t.Year()
	switch {
	case month == 12:
		return "WINTER", year + 1
	case month <= 2:
		return "WINTER", year
	case month <= 5:
		return "SPRING", year
	case month <= 8:
		return "SUMMER", year
	default:
		return "FALL", year
	}
}
