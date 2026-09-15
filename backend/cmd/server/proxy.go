package main

import (
	"bufio"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var httpClient = &http.Client{}

// handleProxy fetches the stream session's URL (or, for a rewritten HLS
// segment/nested-playlist reference, the URL in the "u" query param) with
// the headers the provider requires, and streams the response back. HLS
// manifests are rewritten in transit so every URI they reference also comes
// back through this proxy.
func handleProxy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := store.get(id)
	if !ok {
		http.Error(w, "stream session not found or expired", http.StatusNotFound)
		return
	}

	target := sess.URL
	if u := r.URL.Query().Get("u"); u != "" {
		decoded, err := decodeTarget(u)
		if err != nil {
			http.Error(w, "invalid target", http.StatusBadRequest)
			return
		}
		target = decoded
	}

	if err := validateTarget(target); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, "bad upstream request", http.StatusBadGateway)
		return
	}
	for k, v := range sess.Headers {
		req.Header.Set(k, v)
	}
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, "upstream fetch failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if isManifest(target, resp.Header.Get("Content-Type")) {
		rewriteAndServeManifest(w, resp, target, id)
		return
	}

	copyPassthroughHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func copyPassthroughHeaders(dst http.Header, src http.Header) {
	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Cache-Control"} {
		if v := src.Get(h); v != "" {
			dst.Set(h, v)
		}
	}
}

func isManifest(target, contentType string) bool {
	if strings.Contains(contentType, "mpegurl") {
		return true
	}
	u, err := url.Parse(target)
	if err != nil {
		return false
	}
	return strings.HasSuffix(strings.ToLower(u.Path), ".m3u8")
}

// rewriteAndServeManifest rewrites every URI an HLS playlist references
// (segments, nested/variant playlists, and #EXT-X-KEY / #EXT-X-MAP URIs) to
// point back at this proxy, so every follow-up fetch also carries the
// upstream's required headers.
func rewriteAndServeManifest(w http.ResponseWriter, resp *http.Response, target, id string) {
	base, err := url.Parse(target)
	if err != nil {
		http.Error(w, "invalid manifest base URL", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(http.StatusOK)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	writer := bufio.NewWriter(w)
	defer writer.Flush()

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "":
			writer.WriteString(line + "\n")
		case strings.HasPrefix(trimmed, "#EXT-X-KEY") || strings.HasPrefix(trimmed, "#EXT-X-MAP"):
			writer.WriteString(rewriteAttrURI(line, base, id) + "\n")
		case strings.HasPrefix(trimmed, "#"):
			writer.WriteString(line + "\n")
		default:
			writer.WriteString(proxyURLFor(base, trimmed, id) + "\n")
		}
	}
}

// rewriteAttrURI rewrites a quoted URI="..." attribute within a tag line.
func rewriteAttrURI(line string, base *url.URL, id string) string {
	const marker = `URI="`
	idx := strings.Index(line, marker)
	if idx == -1 {
		return line
	}
	start := idx + len(marker)
	end := strings.Index(line[start:], `"`)
	if end == -1 {
		return line
	}
	original := line[start : start+end]
	rewritten := proxyURLFor(base, original, id)
	return line[:start] + rewritten + line[start+end:]
}

func proxyURLFor(base *url.URL, ref string, id string) string {
	resolved, err := base.Parse(ref)
	if err != nil {
		return ref
	}
	return "/api/proxy/" + id + "?u=" + encodeTarget(resolved.String())
}

func encodeTarget(u string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(u))
}

func decodeTarget(enc string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func validateTarget(target string) error {
	u, err := url.Parse(target)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errInvalidScheme
	}
	return nil
}
