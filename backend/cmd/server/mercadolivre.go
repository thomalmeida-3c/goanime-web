package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

// mercadoLivreSearchURL requires a Bearer access token now — see ml_oauth.go
// for the one-time OAuth setup this needs (their old open public search is
// gone, confirmed by hand: both /sites/MLB/search and /products/search
// answer 403 without a token).
const mercadoLivreSearchURL = "https://api.mercadolibre.com/sites/MLB/search"

var mercadoLivreClient = &http.Client{Timeout: 10 * time.Second}

type Product struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Price     float64 `json:"price"`
	Thumbnail string  `json:"thumbnail"`
	URL       string  `json:"url"`
}

type mlSearchResponse struct {
	Results []struct {
		ID        string  `json:"id"`
		Title     string  `json:"title"`
		Price     float64 `json:"price"`
		Thumbnail string  `json:"thumbnail"`
		Permalink string  `json:"permalink"`
	} `json:"results"`
}

// searchMercadoLivre returns (nil, nil) — not an error — when the one-time
// OAuth consent was never done, so callers (handleProducts) can tell "not
// connected yet" apart from a real upstream failure.
func searchMercadoLivre(ctx context.Context, query string, limit int) ([]Product, error) {
	token, err := getValidMLAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, nil
	}

	u := mercadoLivreSearchURL + "?q=" + url.QueryEscape(query) + fmt.Sprintf("&limit=%d", limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := mercadoLivreClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mercado livre search: status %d", resp.StatusCode)
	}

	var parsed mlSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	products := make([]Product, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		products = append(products, Product{
			ID:        r.ID,
			Title:     r.Title,
			Price:     r.Price,
			Thumbnail: r.Thumbnail,
			URL:       buildAffiliateLink(r.Permalink),
		})
	}
	return products, nil
}

// buildAffiliateLink wraps a Mercado Livre product URL with this project's
// affiliate tag, once it has one (ML_AFFILIATE_TAG unset today — no
// affiliate account yet). Pass-through until then; the exact query
// param/link format their affiliate program expects isn't guessed here —
// fill it in against their real docs once the account exists.
func buildAffiliateLink(permalink string) string {
	tag := os.Getenv("ML_AFFILIATE_TAG")
	if tag == "" {
		return permalink
	}
	sep := "?"
	if containsQuery(permalink) {
		sep = "&"
	}
	return permalink + sep + "matt_tool=" + url.QueryEscape(tag)
}

func containsQuery(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && u.RawQuery != ""
}
