package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LibraryItem is one saved anime or manga in a user's list.
type LibraryItem struct {
	Kind     string    `json:"kind"` // "anime" | "manga"
	Title    string    `json:"title"`
	ImageURL string    `json:"imageUrl,omitempty"`
	RefID    string    `json:"refId"` // anime.URL or manga.id
	Source   string    `json:"source"`
	AddedAt  time.Time `json:"addedAt"`
}

// ChapterProgress is the last chapter a user opened for one manga — "where
// you stopped", not a full read-position tracker (no per-page offset).
type ChapterProgress struct {
	ChapterID string    `json:"chapterId"`
	Chapter   string    `json:"chapter"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// User is keyed by email — there's no password or verification (a deliberate
// MVP choice: this app has no email-sending infrastructure, so "login" is
// just telling us who you are). Good enough for a personal watch/read list,
// not a real account system.
type User struct {
	Email   string        `json:"email"`
	Library []LibraryItem `json:"library"`
	// Progress is keyed by "source|mangaId" — same composite identity
	// LibraryItem uses (a manga id alone isn't unique across sources).
	Progress map[string]ChapterProgress `json:"progress,omitempty"`
}

func progressKey(source, mangaID string) string {
	return source + "|" + mangaID
}

// userStore is a tiny JSON-file-backed persistence layer. Named to avoid
// colliding with the package-level `store` var (the stream-session cache in
// handlers.go — an unrelated, pre-existing "store"). The app's first need
// for anything to survive a restart didn't justify pulling in a SQL driver —
// a mutex-guarded map flushed to disk after every write is simpler, and at
// personal-app scale (a handful of users, a few hundred library items) the
// full-file rewrite on each save is negligible.
type userStore struct {
	mu    sync.Mutex
	path  string
	users map[string]*User
}

func newUserStore(dataDir string) (*userStore, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	s := &userStore{
		path:  filepath.Join(dataDir, "users.json"),
		users: make(map[string]*User),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *userStore) load() error {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var users map[string]*User
	if err := json.Unmarshal(data, &users); err != nil {
		return err
	}
	s.users = users
	return nil
}

// saveLocked writes the full user map to disk. Caller must hold s.mu.
func (s *userStore) saveLocked() error {
	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *userStore) getOrCreateUser(email string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u, ok := s.users[email]; ok {
		return u, nil
	}
	u := &User{Email: email, Library: []LibraryItem{}}
	s.users[email] = u
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *userStore) listLibrary(email string) ([]LibraryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[email]
	if !ok {
		return []LibraryItem{}, nil
	}
	return u.Library, nil
}

func (s *userStore) addLibraryItem(email string, item LibraryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[email]
	if !ok {
		u = &User{Email: email}
		s.users[email] = u
	}
	for _, existing := range u.Library {
		if existing.Kind == item.Kind && existing.RefID == item.RefID {
			return nil // already saved
		}
	}
	item.AddedAt = time.Now()
	u.Library = append(u.Library, item)
	return s.saveLocked()
}

func (s *userStore) removeLibraryItem(email, kind, refID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[email]
	if !ok {
		return nil
	}
	out := u.Library[:0]
	for _, existing := range u.Library {
		if existing.Kind == kind && existing.RefID == refID {
			continue
		}
		out = append(out, existing)
	}
	u.Library = out
	return s.saveLocked()
}

func (s *userStore) setProgress(email, source, mangaID, chapterID, chapter string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[email]
	if !ok {
		u = &User{Email: email}
		s.users[email] = u
	}
	if u.Progress == nil {
		u.Progress = make(map[string]ChapterProgress)
	}
	u.Progress[progressKey(source, mangaID)] = ChapterProgress{
		ChapterID: chapterID,
		Chapter:   chapter,
		UpdatedAt: time.Now(),
	}
	return s.saveLocked()
}

// getProgress's second return is false when the user has never opened a
// chapter of this manga — not an error, just nothing to resume.
func (s *userStore) getProgress(email, source, mangaID string) (ChapterProgress, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[email]
	if !ok {
		return ChapterProgress{}, false
	}
	p, ok := u.Progress[progressKey(source, mangaID)]
	return p, ok
}

// listProgress returns every manga's progress for a user, keyed the same way
// as the internal map ("source|mangaId") — the profile view matches this
// against library items without one request per saved manga.
func (s *userStore) listProgress(email string) (map[string]ChapterProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[email]
	if !ok || u.Progress == nil {
		return map[string]ChapterProgress{}, nil
	}
	out := make(map[string]ChapterProgress, len(u.Progress))
	for k, v := range u.Progress {
		out[k] = v
	}
	return out, nil
}
