package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// mangalivre.blog — a second, unrelated PT-BR manga aggregator (not the
// Madara WordPress theme mangalivre.to runs; a custom theme with its own
// class names). Added as a real second source — not a MangaLivre.to
// replacement — because it carries titles/chapters that .to doesn't (the
// user's own example: Vinland Saga). Same shape as mangalivre.go's
// functions (search/popular/latest/chapters/pages) so manga_handlers.go can
// fan both sources out in parallel and merge the results.
const mangaLivreBlogBase = "https://mangalivre.blog"

var mangaLivreBlogClient = &http.Client{Timeout: 15 * time.Second}

func mangaLivreBlogGet(ctx context.Context, rawURL string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; goanime-web/0.1)")

	resp, err := mangaLivreBlogClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("mangalivre.blog %s: status %d: %s", rawURL, resp.StatusCode, string(body))
	}
	return goquery.NewDocumentFromReader(resp.Body)
}

// parseMangaLivreBlogSearchCards covers the search results page (?s=) —
// .manga-card, with dedicated -link/-title/-cover-img child classes.
func parseMangaLivreBlogSearchCards(doc *goquery.Document, limit int) []MangaItem {
	items := make([]MangaItem, 0) // not nil — see mangalivre.go's note on JSON null
	seen := make(map[string]bool)
	doc.Find(".manga-card").EachWithBreak(func(_ int, card *goquery.Selection) bool {
		if len(items) >= limit {
			return false
		}
		link := card.Find("a.manga-card-link").First()
		href, _ := link.Attr("href")
		title := strings.TrimSpace(card.Find(".manga-card-title").First().Text())
		if href == "" || title == "" || seen[href] {
			return true
		}
		seen[href] = true

		cover, _ := card.Find("img.manga-cover-img").First().Attr("src")

		items = append(items, MangaItem{
			ID:       href,
			Title:    title,
			CoverURL: cover,
			Source:   "MangaLivre.blog",
		})
		return true
	})
	return items
}

// parseMangaLivreBlogCatalogCards covers the /manga/ listing (?ordem=) —
// a completely different template from the search page (.home-manga-card,
// no shared class names), same site otherwise.
func parseMangaLivreBlogCatalogCards(doc *goquery.Document, limit int) []MangaItem {
	items := make([]MangaItem, 0)
	seen := make(map[string]bool)
	doc.Find("article.home-manga-card").EachWithBreak(func(_ int, card *goquery.Selection) bool {
		if len(items) >= limit {
			return false
		}
		href, _ := card.Find("a.home-manga-cover").First().Attr("href")
		title := strings.TrimSpace(card.Find(".home-card-body h3 a").First().Text())
		if href == "" || title == "" || seen[href] {
			return true
		}
		seen[href] = true

		cover, _ := card.Find("a.home-manga-cover img").First().Attr("src")

		items = append(items, MangaItem{
			ID:       href,
			Title:    title,
			CoverURL: cover,
			Source:   "MangaLivre.blog",
		})
		return true
	})
	return items
}

func searchMangaLivreBlog(ctx context.Context, query string, limit int) ([]MangaItem, error) {
	u := mangaLivreBlogBase + "/?s=" + url.QueryEscape(query)
	doc, err := mangaLivreBlogGet(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseMangaLivreBlogSearchCards(doc, limit), nil
}

func fetchMangaLivreBlogPopular(ctx context.Context, limit int) ([]MangaItem, error) {
	u := mangaLivreBlogBase + "/manga/?ordem=popular"
	doc, err := mangaLivreBlogGet(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseMangaLivreBlogCatalogCards(doc, limit), nil
}

func fetchMangaLivreBlogLatest(ctx context.Context, limit int) ([]MangaItem, error) {
	u := mangaLivreBlogBase + "/manga/?ordem=recentes"
	doc, err := mangaLivreBlogGet(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseMangaLivreBlogCatalogCards(doc, limit), nil
}

// fetchMangaLivreBlogChapters reads the manga page's chapter list — like
// mangalivre.to, fully server-rendered on one page (no pagination), newest
// first in the DOM so reversed to reading order. Reuses mangaChapterNumRe
// (mangalivre.go) since chapter labels here sometimes carry a subtitle
// after the number ("Capítulo 218: Viagem de Mil anos, Parte 27").
func fetchMangaLivreBlogChapters(ctx context.Context, mangaURL string) ([]ChapterItem, error) {
	doc, err := mangaLivreBlogGet(ctx, mangaURL)
	if err != nil {
		return nil, err
	}

	items := make([]ChapterItem, 0)
	seen := make(map[string]bool)
	doc.Find("a.chapter-link").Each(func(_ int, a *goquery.Selection) {
		href, ok := a.Attr("href")
		if !ok || href == "" || seen[href] {
			return
		}
		seen[href] = true
		text := strings.TrimSpace(a.Find(".chapter-number").First().Text())
		num := mangaChapterNumRe.FindString(text)
		if num == "" {
			num = text
		}
		items = append(items, ChapterItem{ID: href, Chapter: num})
	})

	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items, nil
}

// resolveMangaLivreBlogChapterPages reads page images straight off the
// chapter page (img.chapter-image, in DOM/reading order) — self-hosted
// static files, no CDN token dance.
func resolveMangaLivreBlogChapterPages(ctx context.Context, chapterURL string) ([]string, error) {
	doc, err := mangaLivreBlogGet(ctx, chapterURL)
	if err != nil {
		return nil, err
	}

	pages := make([]string, 0)
	doc.Find("img.chapter-image").Each(func(_ int, img *goquery.Selection) {
		src, ok := img.Attr("src")
		if !ok || strings.TrimSpace(src) == "" {
			src, _ = img.Attr("data-src")
		}
		if src = strings.TrimSpace(src); src != "" {
			pages = append(pages, src)
		}
	})
	return pages, nil
}
