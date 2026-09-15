package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// streamSession holds the upstream stream URL and the HTTP headers required
// to fetch it (e.g. Referer/User-Agent), keyed by an opaque id handed to the
// frontend. Providers reject requests without these headers, so the browser
// can't hit the stream directly — it has to go through our proxy, which
// attaches them server-side.
type streamSession struct {
	URL     string
	Headers map[string]string
	Created time.Time
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]streamSession
}

func newSessionStore() *sessionStore {
	s := &sessionStore{sessions: make(map[string]streamSession)}
	go s.janitor()
	return s
}

func (s *sessionStore) put(sess streamSession) string {
	id := randomID()
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return id
}

func (s *sessionStore) get(id string) (streamSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

// janitor evicts sessions older than two hours so long-running dev servers
// don't accumulate memory; there is no persistence requirement for an MVP.
func (s *sessionStore) janitor() {
	for range time.Tick(10 * time.Minute) {
		cutoff := time.Now().Add(-2 * time.Hour)
		s.mu.Lock()
		for id, sess := range s.sessions {
			if sess.Created.Before(cutoff) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
