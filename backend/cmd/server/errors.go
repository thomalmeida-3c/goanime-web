package main

import "errors"

var (
	errQueryRequired                  = errors.New("query parameter 'q' is required")
	errURLAndSourceRequired           = errors.New("query parameters 'url' and 'source' are required")
	errEpisodeAndSourceRequired       = errors.New("query parameters 'episodeUrl' and 'source' are required")
	errInvalidScheme                  = errors.New("proxy target must be http or https")
	errAnimeNameAndEpisodeNumRequired = errors.New("query parameters 'animeName' and 'episodeNum' are required")
)
