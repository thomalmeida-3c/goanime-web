package main

import "errors"

var (
	errQueryRequired                  = errors.New("query parameter 'q' is required")
	errURLAndSourceRequired           = errors.New("query parameters 'url' and 'source' are required")
	errEpisodeAndSourceRequired       = errors.New("query parameters 'episodeUrl' and 'source' are required")
	errInvalidScheme                  = errors.New("proxy target must be http or https")
	errAnimeNameAndEpisodeNumRequired = errors.New("query parameters 'animeName' and 'episodeNum' are required")
	errMangaIDRequired                = errors.New("query parameter 'id' is required")
	errChapterIDRequired              = errors.New("query parameter 'id' is required")
	errInvalidEmail                   = errors.New("invalid email")
	errEmailRequired                  = errors.New("query parameter 'email' is required")
	errLibraryItemInvalid             = errors.New("'email', 'item.kind' and 'item.refId' (or 'kind'/'refId' for delete) are required")
	errMLNotConfigured                = errors.New("ML_CLIENT_ID, ML_CLIENT_SECRET and ML_REDIRECT_URI must be set first")
	errProgressInvalid                = errors.New("'email', 'source' and 'mangaId' are required")
)
