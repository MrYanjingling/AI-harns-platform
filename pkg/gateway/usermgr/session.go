package usermgr

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type sessionStore struct {
	mu    sync.RWMutex
	items map[string]*Session
}

func newSessionStore() *sessionStore {
	return &sessionStore{items: make(map[string]*Session)}
}

func (s *sessionStore) Add(session *Session) {
	if s == nil || session == nil || session.Token == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[session.Token] = session
}

func (s *sessionStore) GetAndTouch(token string, now time.Time) (*Session, bool) {
	if s == nil || token == "" {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.items[token]
	if !ok {
		return nil, false
	}
	if !now.Before(session.ExpiresAt) || !now.Before(session.AbsoluteExpiresAt) {
		delete(s.items, token)
		return nil, false
	}
	next := now.Add(DefaultSessionTTL)
	if next.After(session.AbsoluteExpiresAt) {
		next = session.AbsoluteExpiresAt
	}
	session.ExpiresAt = next
	copy := *session
	return &copy, true
}

func (s *sessionStore) Delete(token string) {
	if s == nil || token == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, token)
}

func (s *sessionStore) DeleteByUser(userID string) {
	if s == nil || userID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, session := range s.items {
		if session.UserID == userID {
			delete(s.items, token)
		}
	}
}

func newToken() (string, error) {
	buf := make([]byte, tokenLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
