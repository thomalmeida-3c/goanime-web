package api

import "github.com/alvarorichard/Goanime/internal/models"

// EnrichAnimeMetadata is enrichAnimeData exported for cmd/server: given just
// Name and URL it looks the title up on AniList (falling back to
// MyAnimeList) and fills in AnilistID/MalID/Details. Needed because AniSkip
// (skip-intro/outro times) is keyed by MAL id, which scraper sources like
// Goyabu never expose on their own — the web backend has to resolve it the
// same way the CLI does before it can ask AniSkip anything.
func EnrichAnimeMetadata(anime *models.Anime) error {
	return enrichAnimeData(anime)
}
