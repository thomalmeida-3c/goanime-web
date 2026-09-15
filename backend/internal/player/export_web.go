package player

// ResolveBloggerStream extracts a Blogger (blogger.com/video.g) video and
// starts the local proxy that streams the underlying googlevideo CDN
// response with Chrome TLS impersonation (the CDN rejects Go's native TLS
// fingerprint). Returns a http://127.0.0.1:<port>/blogger_proxy URL.
//
// Exported for non-CLI consumers (cmd/server) that need the same resolution
// path the CLI hands to mpv. Note: the underlying proxy is a package-level
// singleton (StopBloggerProxy stops the previous one), so only one resolved
// Blogger stream is active at a time — fine for a single local dev server,
// not for concurrent multi-user playback.
func ResolveBloggerStream(bloggerURL string) (string, error) {
	return extractBloggerVideoURL(bloggerURL)
}
