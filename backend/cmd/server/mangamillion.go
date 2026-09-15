package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

// Shueisha's own official reading platform (MANGA MILLION) — unlike every
// other manga source here, this is the actual rights holder's site, not a
// fan aggregator. Its catalog API (/api/manga_list) is public and
// unauthenticated, but it answers with raw Protocol Buffers and Shueisha
// hasn't published the .proto schema. Reading chapters requires an
// Access-Token that's minted client-side by their JS with no visible
// network round-trip (not forgeable with an arbitrary value — tried,
// 403s) — reverse-engineering that felt like the wrong thing to spend
// effort circumventing on the publisher's own access gate, so this only
// surfaces the catalog for discovery; opening a title links out to
// mangamillion.shueisha.co.jp itself to actually read (page.tsx checks
// `source === "MangaMillion"` and opens externally instead of routing into
// our own chapters/reader views, which don't exist for this source).
const mangaMillionBase = "https://mangamillion.shueisha.co.jp"
const mangaMillionAPIBase = "https://api.mangamillion.shueisha.co.jp"

var mangaMillionClient = &http.Client{Timeout: 20 * time.Second}

type mangaMillionTitle struct {
	ID      int64
	Cover   string
	Title   string
	Author  string
	Views   int64
	Updated int64
}

var (
	mangaMillionCacheMu sync.Mutex
	mangaMillionCache   []mangaMillionTitle
	mangaMillionCacheAt time.Time
)

const mangaMillionCacheTTL = time.Hour

// getMangaMillionCatalog caches the full 394-title catalog (one ~130KB
// fetch) so popular/latest/search all reuse it instead of hitting the API
// per request — same idea as mangaHomeCache, just source-scoped since this
// one source backs three different views.
func getMangaMillionCatalog(ctx context.Context) ([]mangaMillionTitle, error) {
	mangaMillionCacheMu.Lock()
	if mangaMillionCache != nil && time.Since(mangaMillionCacheAt) < mangaMillionCacheTTL {
		cached := mangaMillionCache
		mangaMillionCacheMu.Unlock()
		return cached, nil
	}
	mangaMillionCacheMu.Unlock()

	items, err := fetchMangaMillionCatalog(ctx)
	if err != nil {
		return nil, err
	}

	mangaMillionCacheMu.Lock()
	mangaMillionCache = items
	mangaMillionCacheAt = time.Now()
	mangaMillionCacheMu.Unlock()
	return items, nil
}

func fetchMangaMillionCatalog(ctx context.Context) ([]mangaMillionTitle, error) {
	u := mangaMillionAPIBase + "/api/manga_list?service_language=pt-BR&avif_enable=true&tag_name=all-titles"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	// The API's CORS policy only allows the mangamillion.shueisha.co.jp
	// origin — that's a browser-side check, not enforced against a
	// server-to-server client, but sending it anyway matches what a real
	// browser would send and costs nothing.
	req.Header.Set("Origin", mangaMillionBase)

	resp, err := mangaMillionClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("mangamillion manga_list: status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseMangaMillionCatalog(data)
}

// parseMangaMillionCatalog hand-decodes the response's wire format — there
// is no published .proto for this API. Field numbers below were found by
// running `protoc --decode_raw` against a captured response: the top-level
// message wraps a repeated field 22.1, and each entry's actual data lives
// one level deeper in field 1 (id, cover, title, author, ..., views,
// updated-at). Unknown fields are skipped via ConsumeFieldValue rather than
// assumed absent, so this keeps working if Shueisha adds fields later —
// but a field *renumbering* would silently break it (no schema to detect
// that against); if this source ever comes back empty, that's the first
// thing to check.
func parseMangaMillionCatalog(data []byte) ([]mangaMillionTitle, error) {
	entries, err := findRepeatedSubmessage(data, 22)
	if err != nil {
		return nil, err
	}
	if len(entries) != 1 {
		return nil, fmt.Errorf("mangamillion: expected exactly one field-22 message, got %d", len(entries))
	}

	titleEntries, err := findRepeatedSubmessage(entries[0], 1)
	if err != nil {
		return nil, err
	}

	items := make([]mangaMillionTitle, 0, len(titleEntries))
	for _, entry := range titleEntries {
		info, err := findRepeatedSubmessage(entry, 1)
		if err != nil || len(info) == 0 {
			continue
		}
		items = append(items, parseMangaMillionInfo(info[0]))
	}
	return items, nil
}

// findRepeatedSubmessage walks a message's top-level fields and returns the
// raw bytes of every occurrence of a given field number that's a
// length-delimited (submessage/string/bytes) value.
func findRepeatedSubmessage(b []byte, field protowire.Number) ([][]byte, error) {
	var out [][]byte
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		b = b[n:]
		if num == field && typ == protowire.BytesType {
			v, n2 := protowire.ConsumeBytes(b)
			if n2 < 0 {
				return nil, protowire.ParseError(n2)
			}
			out = append(out, v)
			b = b[n2:]
			continue
		}
		n3 := protowire.ConsumeFieldValue(num, typ, b)
		if n3 < 0 {
			return nil, protowire.ParseError(n3)
		}
		b = b[n3:]
	}
	return out, nil
}

func parseMangaMillionInfo(b []byte) mangaMillionTitle {
	var t mangaMillionTitle
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return t
		}
		b = b[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n2 := protowire.ConsumeVarint(b)
			if n2 < 0 {
				return t
			}
			t.ID = int64(v)
			b = b[n2:]
		case num == 2 && typ == protowire.BytesType:
			v, n2 := protowire.ConsumeBytes(b)
			if n2 < 0 {
				return t
			}
			t.Cover = string(v)
			b = b[n2:]
		case num == 3 && typ == protowire.BytesType:
			v, n2 := protowire.ConsumeBytes(b)
			if n2 < 0 {
				return t
			}
			t.Title = string(v)
			b = b[n2:]
		case num == 4 && typ == protowire.BytesType:
			v, n2 := protowire.ConsumeBytes(b)
			if n2 < 0 {
				return t
			}
			t.Author = string(v)
			b = b[n2:]
		case num == 7 && typ == protowire.VarintType:
			v, n2 := protowire.ConsumeVarint(b)
			if n2 < 0 {
				return t
			}
			t.Views = int64(v)
			b = b[n2:]
		case num == 9 && typ == protowire.VarintType:
			v, n2 := protowire.ConsumeVarint(b)
			if n2 < 0 {
				return t
			}
			t.Updated = int64(v)
			b = b[n2:]
		default:
			n3 := protowire.ConsumeFieldValue(num, typ, b)
			if n3 < 0 {
				return t
			}
			b = b[n3:]
		}
	}
	return t
}

// mangaMillionToMangaItem's ID is the full external title-page URL (not a
// bare numeric id, unlike everything else this parses off it) — this
// source has no chapters/pages endpoint of its own, so the frontend opens
// it directly instead of routing into our reader.
func mangaMillionToMangaItem(t mangaMillionTitle) MangaItem {
	return MangaItem{
		ID:          mangaMillionBase + "/pt-BR/title/" + strconv.FormatInt(t.ID, 10),
		Title:       t.Title,
		Description: t.Author,
		CoverURL:    t.Cover,
		Source:      "MangaMillion",
	}
}

func fetchMangaMillionPopular(ctx context.Context, limit int) ([]MangaItem, error) {
	all, err := getMangaMillionCatalog(ctx)
	if err != nil {
		return nil, err
	}
	sorted := append([]mangaMillionTitle(nil), all...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Views > sorted[j].Views })
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return toMangaMillionItems(sorted), nil
}

func fetchMangaMillionLatest(ctx context.Context, limit int) ([]MangaItem, error) {
	all, err := getMangaMillionCatalog(ctx)
	if err != nil {
		return nil, err
	}
	sorted := append([]mangaMillionTitle(nil), all...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Updated > sorted[j].Updated })
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return toMangaMillionItems(sorted), nil
}

// searchMangaMillion filters the cached catalog locally (title substring
// match) rather than hitting a separate search endpoint — 394 titles is
// small enough that this is instant, and it's one less protobuf shape to
// reverse-engineer.
func searchMangaMillion(ctx context.Context, query string, limit int) ([]MangaItem, error) {
	all, err := getMangaMillionCatalog(ctx)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	var matched []mangaMillionTitle
	for _, t := range all {
		if strings.Contains(strings.ToLower(t.Title), q) {
			matched = append(matched, t)
			if len(matched) >= limit {
				break
			}
		}
	}
	return toMangaMillionItems(matched), nil
}

func toMangaMillionItems(titles []mangaMillionTitle) []MangaItem {
	items := make([]MangaItem, 0, len(titles))
	for _, t := range titles {
		items = append(items, mangaMillionToMangaItem(t))
	}
	return items
}
