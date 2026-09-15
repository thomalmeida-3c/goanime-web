package main

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// users persists the (email-only, no password) accounts and their saved
// anime/manga library. No cookie/session: the email is the identity, passed
// on every request — same pattern as `source` for manga endpoints.
var users = mustNewUserStore()

func mustNewUserStore() *userStore {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	s, err := newUserStore(dataDir)
	if err != nil {
		panic("failed to initialize user store: " + err.Error())
	}
	return s
}

func registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", handleLogin)
	mux.HandleFunc("GET /api/library", handleGetLibrary)
	mux.HandleFunc("POST /api/library", handlePostLibrary)
	mux.HandleFunc("DELETE /api/library", handleDeleteLibrary)
	mux.HandleFunc("GET /api/manga/progress", handleGetProgress)
	mux.HandleFunc("POST /api/manga/progress", handlePostProgress)
	mux.HandleFunc("GET /api/manga/progress/all", handleListProgress)
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// handleLogin is deliberately not real authentication — there's no password,
// no verification email. It just validates the format and creates the
// account on first use, so the frontend has something to remember the user
// by (localStorage) and send back on every library call. Good enough for a
// personal watch/read list; not a security boundary.
func handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	email := normalizeEmail(body.Email)
	if !emailRe.MatchString(email) {
		writeError(w, http.StatusBadRequest, errInvalidEmail)
		return
	}

	user, err := users.getOrCreateUser(email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func handleGetLibrary(w http.ResponseWriter, r *http.Request) {
	email := normalizeEmail(r.URL.Query().Get("email"))
	if email == "" {
		writeError(w, http.StatusBadRequest, errEmailRequired)
		return
	}

	items, err := users.listLibrary(email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func handlePostLibrary(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string      `json:"email"`
		Item  LibraryItem `json:"item"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	email := normalizeEmail(body.Email)
	if email == "" || body.Item.RefID == "" || body.Item.Kind == "" {
		writeError(w, http.StatusBadRequest, errLibraryItemInvalid)
		return
	}

	if err := users.addLibraryItem(email, body.Item); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleDeleteLibrary(w http.ResponseWriter, r *http.Request) {
	email := normalizeEmail(r.URL.Query().Get("email"))
	kind := r.URL.Query().Get("kind")
	refID := r.URL.Query().Get("refId")
	if email == "" || kind == "" || refID == "" {
		writeError(w, http.StatusBadRequest, errLibraryItemInvalid)
		return
	}

	if err := users.removeLibraryItem(email, kind, refID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleGetProgress always returns 200 — {"found": false} isn't an error,
// it just means the user hasn't opened a chapter of this manga yet (same
// "empty state, not a failure" pattern as /api/products' {connected:false}).
func handleGetProgress(w http.ResponseWriter, r *http.Request) {
	email := normalizeEmail(r.URL.Query().Get("email"))
	source := r.URL.Query().Get("source")
	mangaID := r.URL.Query().Get("mangaId")
	if email == "" || source == "" || mangaID == "" {
		writeError(w, http.StatusBadRequest, errProgressInvalid)
		return
	}

	p, found := users.getProgress(email, source, mangaID)
	writeJSON(w, http.StatusOK, struct {
		Found bool `json:"found"`
		ChapterProgress
	}{Found: found, ChapterProgress: p})
}

func handlePostProgress(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email     string `json:"email"`
		Source    string `json:"source"`
		MangaID   string `json:"mangaId"`
		ChapterID string `json:"chapterId"`
		Chapter   string `json:"chapter"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	email := normalizeEmail(body.Email)
	if email == "" || body.Source == "" || body.MangaID == "" || body.ChapterID == "" {
		writeError(w, http.StatusBadRequest, errProgressInvalid)
		return
	}

	if err := users.setProgress(email, body.Source, body.MangaID, body.ChapterID, body.Chapter); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleListProgress powers the profile page: one call instead of one
// /api/manga/progress per saved manga.
func handleListProgress(w http.ResponseWriter, r *http.Request) {
	email := normalizeEmail(r.URL.Query().Get("email"))
	if email == "" {
		writeError(w, http.StatusBadRequest, errEmailRequired)
		return
	}

	m, err := users.listProgress(email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}
