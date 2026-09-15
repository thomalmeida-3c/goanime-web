package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// MangaDex (https://api.mangadex.org) is a public, keyless manga API — no
// scraper needed, unlike the anime sources. Manga pages are served straight
// from MangaDex's own CDN (MangaDex@Home) directly to the browser, so unlike
// video we never need to proxy them ourselves.

const (
	mangadexBase    = "https://api.mangadex.org"
	mangadexUploads = "https://uploads.mangadex.org"
)

var mangadexClient = &http.Client{Timeout: 15 * time.Second}

// titlePriority picks a Portuguese title/description when MangaDex has one,
// falling back to English/Japanese romaji, then whatever else is present —
// most entries only ship a subset of locales.
var titlePriority = []string{"pt-br", "pt", "en", "ja-ro", "ja"}

func pickLocalized(m map[string]string) string {
	for _, lang := range titlePriority {
		if v, ok := m[lang]; ok && v != "" {
			return v
		}
	}
	for _, v := range m {
		if v != "" {
			return v
		}
	}
	return ""
}

// MangaItem is our public shape for a manga (mirrors HomeItem's style: flat,
// camelCase, only what the frontend renders).
type MangaItem struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description,omitempty"`
	CoverURL      string   `json:"coverUrl,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	ContentRating string   `json:"contentRating,omitempty"`
	Year          int      `json:"year,omitempty"`
}

type ChapterItem struct {
	ID      string `json:"id"`
	Chapter string `json:"chapter"`
	Title   string `json:"title,omitempty"`
}

type mdManga struct {
	ID         string `json:"id"`
	Attributes struct {
		Title         map[string]string `json:"title"`
		Description   map[string]string `json:"description"`
		ContentRating string            `json:"contentRating"`
		Year          int               `json:"year"`
		Tags          []struct {
			Attributes struct {
				Name map[string]string `json:"name"`
			} `json:"attributes"`
		} `json:"tags"`
	} `json:"attributes"`
	Relationships []struct {
		Type       string `json:"type"`
		Attributes struct {
			FileName string `json:"fileName"`
		} `json:"attributes"`
	} `json:"relationships"`
}

type mdMangaListResponse struct {
	Data []mdManga `json:"data"`
}

type mdChapter struct {
	ID         string `json:"id"`
	Attributes struct {
		Chapter            string `json:"chapter"`
		Title              string `json:"title"`
		TranslatedLanguage string `json:"translatedLanguage"`
	} `json:"attributes"`
}

type mdChapterListResponse struct {
	Data []mdChapter `json:"data"`
}

type mdAtHomeResponse struct {
	BaseURL string `json:"baseUrl"`
	Chapter struct {
		Hash string   `json:"hash"`
		Data []string `json:"data"`
	} `json:"chapter"`
}

func mangadexGet(ctx context.Context, path string, query url.Values, out any) error {
	u := mangadexBase + path
	if query != nil {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "goanime-web/0.1 (+https://github.com/alvarorichard/GoAnime)")

	resp, err := mangadexClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("mangadex %s: status %d: %s", path, resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// contentRatingParams excludes erotica/pornographic — this is a general
// anime app's default catalog, not an 18+ one.
func contentRatingParams(v url.Values) {
	v.Add("contentRating[]", "safe")
	v.Add("contentRating[]", "suggestive")
}

func toMangaItems(resources []mdManga) []MangaItem {
	items := make([]MangaItem, 0, len(resources))
	for _, r := range resources {
		item := MangaItem{
			ID:            r.ID,
			Title:         pickLocalized(r.Attributes.Title),
			Description:   pickLocalized(r.Attributes.Description),
			ContentRating: r.Attributes.ContentRating,
			Year:          r.Attributes.Year,
		}
		for _, tag := range r.Attributes.Tags {
			if name := pickLocalized(tag.Attributes.Name); name != "" {
				item.Tags = append(item.Tags, name)
			}
		}
		for _, rel := range r.Relationships {
			if rel.Type == "cover_art" && rel.Attributes.FileName != "" {
				item.CoverURL = fmt.Sprintf("%s/covers/%s/%s.512.jpg", mangadexUploads, r.ID, rel.Attributes.FileName)
			}
		}
		items = append(items, item)
	}
	return items
}

// fetchMangaDexPopular lists manga ordered by follower count, restricted to
// titles that actually have Portuguese chapters — there's no point
// surfacing a manga on the home page the user can't read.
func fetchMangaDexPopular(ctx context.Context, limit int) ([]MangaItem, error) {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(limit))
	v.Add("order[followedCount]", "desc")
	v.Add("availableTranslatedLanguage[]", "pt-br")
	v.Add("includes[]", "cover_art")
	contentRatingParams(v)

	var resp mdMangaListResponse
	if err := mangadexGet(ctx, "/manga", v, &resp); err != nil {
		return nil, err
	}
	return toMangaItems(resp.Data), nil
}

func searchMangaDex(ctx context.Context, query string, limit int) ([]MangaItem, error) {
	v := url.Values{}
	v.Set("title", query)
	v.Set("limit", strconv.Itoa(limit))
	v.Add("availableTranslatedLanguage[]", "pt-br")
	v.Add("includes[]", "cover_art")
	contentRatingParams(v)

	var resp mdMangaListResponse
	if err := mangadexGet(ctx, "/manga", v, &resp); err != nil {
		return nil, err
	}
	return toMangaItems(resp.Data), nil
}

// fetchMangaChapters lists a manga's Portuguese chapters in reading order.
// Multiple scanlation groups often translate the same chapter; this keeps
// only the first one MangaDex returns per chapter number rather than
// showing duplicates.
func fetchMangaChapters(ctx context.Context, mangaID string) ([]ChapterItem, error) {
	v := url.Values{}
	v.Add("translatedLanguage[]", "pt-br")
	v.Add("order[chapter]", "asc")
	v.Set("limit", "500")

	var resp mdChapterListResponse
	if err := mangadexGet(ctx, "/manga/"+mangaID+"/feed", v, &resp); err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(resp.Data))
	items := make([]ChapterItem, 0, len(resp.Data))
	for _, c := range resp.Data {
		if seen[c.Attributes.Chapter] {
			continue
		}
		seen[c.Attributes.Chapter] = true
		items = append(items, ChapterItem{
			ID:      c.ID,
			Chapter: c.Attributes.Chapter,
			Title:   c.Attributes.Title,
		})
	}
	return items, nil
}

// resolveChapterPages asks MangaDex which CDN node to read this chapter's
// pages from (the "MangaDex@Home" network) and builds the page image URLs.
// This must be called fresh per read, right before display — it's not
// something to cache long-term.
func resolveChapterPages(ctx context.Context, chapterID string) ([]string, error) {
	var resp mdAtHomeResponse
	if err := mangadexGet(ctx, "/at-home/server/"+chapterID, nil, &resp); err != nil {
		return nil, err
	}

	pages := make([]string, 0, len(resp.Chapter.Data))
	for _, filename := range resp.Chapter.Data {
		pages = append(pages, fmt.Sprintf("%s/data/%s/%s", resp.BaseURL, resp.Chapter.Hash, filename))
	}
	return pages, nil
}
