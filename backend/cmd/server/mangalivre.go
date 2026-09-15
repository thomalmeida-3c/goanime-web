package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// mangalivre.to runs the Madara WordPress theme, common to a lot of PT-BR
// manga aggregators. Unlike the anime sources, it needs no Cloudflare
// solve, no session/token — plain HTTP GET + HTML parsing (goquery, same as
// GoAnime's own AnimeFire/Goyabu scrapers), and chapter pages serve their
// images from the same origin with no special headers required. Its big
// advantage over MangaDex: full chapter runs for officially-licensed titles
// (e.g. Boruto, Jujutsu Kaisen) that MangaDex can only link out to (and
// often no longer even that, once the publisher pulls the simulpub).
const mangaLivreBase = "https://mangalivre.to"

var mangaLivreClient = &http.Client{Timeout: 15 * time.Second}

func mangaLivreGet(ctx context.Context, rawURL string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; goanime-web/0.1)")

	resp, err := mangaLivreClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("mangalivre %s: status %d: %s", rawURL, resp.StatusCode, string(body))
	}
	return goquery.NewDocumentFromReader(resp.Body)
}

// searchMangaLivre uses the theme's plain WordPress search (?s=) — no AJAX
// endpoint needed, the results page already renders full manga cards.
func searchMangaLivre(ctx context.Context, query string, limit int) ([]MangaItem, error) {
	u := mangaLivreBase + "/?s=" + url.QueryEscape(query)
	doc, err := mangaLivreGet(ctx, u)
	if err != nil {
		return nil, err
	}

	var items []MangaItem
	seen := make(map[string]bool)
	doc.Find(".c-tabs-item__content").EachWithBreak(func(_ int, card *goquery.Selection) bool {
		if len(items) >= limit {
			return false
		}
		link := card.Find(".post-title a").First()
		href, _ := link.Attr("href")
		title := strings.TrimSpace(link.Text())
		if href == "" || title == "" || seen[href] {
			return true
		}
		seen[href] = true

		cover, _ := card.Find(".tab-thumb img").Attr("src")

		items = append(items, MangaItem{
			ID:       href,
			Title:    title,
			CoverURL: cover,
			Source:   "MangaLivre",
		})
		return true
	})
	return items, nil
}

// mangaChapterNumRe pulls the leading number out of a chapter link's label
// ("Capitulo 36" -> "36"), matching the plain numeric Chapter field the
// frontend already expects from MangaDex.
var mangaChapterNumRe = regexp.MustCompile(`\d+(\.\d+)?`)

// fetchMangaLivreChapters reads the manga page's chapter list, which —
// unlike MangaDex's per-chapter-per-group feed — is server-rendered in full
// on a single page, no pagination to chase. Newest-first in the DOM, so
// it's reversed to match reading order.
func fetchMangaLivreChapters(ctx context.Context, mangaURL string) ([]ChapterItem, error) {
	doc, err := mangaLivreGet(ctx, mangaURL)
	if err != nil {
		return nil, err
	}

	var items []ChapterItem
	doc.Find(".listing-chapters-wrap a").Each(func(_ int, a *goquery.Selection) {
		href, ok := a.Attr("href")
		if !ok || href == "" {
			return
		}
		text := strings.TrimSpace(a.Text())
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

// resolveMangaLivreChapterPages reads the page images straight off the
// chapter page — no CDN token dance like MangaDex's at-home/server, the
// images are just self-hosted static files.
func resolveMangaLivreChapterPages(ctx context.Context, chapterURL string) ([]string, error) {
	doc, err := mangaLivreGet(ctx, chapterURL)
	if err != nil {
		return nil, err
	}

	var pages []string
	doc.Find("img.wp-manga-chapter-img").Each(func(_ int, img *goquery.Selection) {
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
