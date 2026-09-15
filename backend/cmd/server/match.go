package main

import (
	"regexp"
	"strings"
	"sync"

	"github.com/alvarorichard/Goanime/pkg/goanime/types"
)

var (
	bracketOrParenRe = regexp.MustCompile(`[\[\(][^\]\)]*[\]\)]`)
	nonAlnumRe       = regexp.MustCompile(`[^a-z0-9 ]+`)
	htmlTagRe        = regexp.MustCompile(`<[^>]*>`)
)

// noiseWords are tokens that show up in scraper-site titles but never in
// AniList's canonical title, so they'd otherwise drag down the match score.
var noiseWords = map[string]bool{
	"pt": true, "br": true, "ptbr": true, "english": true,
	"dublado": true, "legendado": true, "dub": true, "leg": true,
}

func stripHTML(s string) string {
	return strings.TrimSpace(htmlTagRe.ReplaceAllString(s, " "))
}

// normalizeTitle lowercases, drops bracketed/parenthesized tags like
// "[PT-BR]" or "(Dublado)", strips punctuation, and removes noise words —
// leaving just the tokens that should line up between an AniList title and a
// scraper-site title for the same show.
func normalizeTitle(s string) string {
	s = strings.ToLower(s)
	s = bracketOrParenRe.ReplaceAllString(s, " ")
	s = nonAlnumRe.ReplaceAllString(s, " ")

	fields := strings.Fields(s)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if !noiseWords[f] {
			out = append(out, f)
		}
	}
	return strings.Join(out, " ")
}

// tokenOverlap scores how much of the shorter token set is covered by the
// longer one, so "sousou no frieren" fully matches inside
// "sousou no frieren beyond journeys end" (season 2's longer title) just as
// well as an exact match.
func tokenOverlap(a, b string) float64 {
	as := strings.Fields(a)
	bs := strings.Fields(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0
	}

	bset := make(map[string]bool, len(bs))
	for _, w := range bs {
		bset[w] = true
	}

	matched := 0
	for _, w := range as {
		if bset[w] {
			matched++
		}
	}

	denom := len(as)
	if len(bs) < denom {
		denom = len(bs)
	}
	return float64(matched) / float64(denom)
}

const matchThreshold = 0.6

// bestMatch picks the search result whose normalized name best overlaps the
// (already normalized) query, requiring a minimum score so an unrelated
// result never gets linked as if it were the AniList title.
//
// tokenOverlap alone ties "Kimetsu no Yaiba" and "Kimetsu no Yaiba: Mugen
// Ressha-hen" at 1.0 (the query is fully contained in both), and the movie
// spinoff would win just by sorting first in the site's search results. Ties
// are broken toward the candidate whose token count is closest to the
// query's, since a bare title match is far more likely to be the main
// series than a same-prefix movie/OVA with an extra subtitle.
func bestMatch(normalizedQuery string, results []*types.Anime) *types.Anime {
	queryTokens := len(strings.Fields(normalizedQuery))

	var best *types.Anime
	bestScore := 0.0
	bestExtra := 0

	for _, r := range results {
		normalized := normalizeTitle(r.Name)
		score := tokenOverlap(normalizedQuery, normalized)
		if score < matchThreshold {
			continue
		}

		extra := len(strings.Fields(normalized)) - queryTokens
		if extra < 0 {
			extra = -extra
		}

		if best == nil || score > bestScore || (score == bestScore && extra < bestExtra) {
			best, bestScore, bestExtra = r, score, extra
		}
	}
	return best
}

// findGoyabuMatch tries each candidate title (romaji, then English) against
// a live Goyabu search and returns the first good match. Goyabu is the only
// source with a fully working search->episodes->stream path today (see
// README), so that's what home-page cards link to.
func findGoyabuMatch(titles ...string) *types.Anime {
	src := types.SourceGoyabu
	seen := make(map[string]bool, len(titles))

	for _, title := range titles {
		title = strings.TrimSpace(title)
		if title == "" || seen[title] {
			continue
		}
		seen[title] = true

		results, err := client.SearchAnime(title, &src)
		if err != nil || len(results) == 0 {
			continue
		}
		if match := bestMatch(normalizeTitle(title), results); match != nil {
			return match
		}
	}
	return nil
}

// matchAll resolves a batch of AniList entries to HomeItems concurrently
// (bounded — each match is a live scrape against Goyabu, so unbounded
// fan-out would hammer the site and our own scraper's rate limiting).
func matchAll(media []anilistMedia) []HomeItem {
	items := make([]HomeItem, len(media))
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup

	for i, m := range media {
		wg.Add(1)
		go func(i int, m anilistMedia) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			items[i] = toHomeItem(m)
		}(i, m)
	}
	wg.Wait()
	return items
}

func toHomeItem(m anilistMedia) HomeItem {
	title := m.Title.Romaji
	if title == "" {
		title = m.Title.English
	}

	item := HomeItem{
		AnilistID:   m.ID,
		Title:       title,
		ImageURL:    m.CoverImage.Large,
		BannerURL:   m.BannerImage,
		Description: stripHTML(m.Description),
		Score:       m.AverageScore,
		Genres:      m.Genres,
	}

	if match := findGoyabuMatch(m.Title.Romaji, m.Title.English); match != nil {
		item.AnimeURL = match.URL
		item.Source = match.Source
	}

	return item
}
