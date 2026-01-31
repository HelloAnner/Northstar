package v3

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type exportDownload struct {
	filePath  string
	year      int
	month     int
	expiresAt time.Time
}

type exportDownloadStore struct {
	mu    sync.Mutex
	items map[string]exportDownload
}

func newExportDownloadStore() *exportDownloadStore {
	return &exportDownloadStore{
		items: make(map[string]exportDownload),
	}
}

func (s *exportDownloadStore) put(filePath string, year, month int, ttl time.Duration) (token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked(time.Now())

	token = newRandomToken(24)
	s.items[token] = exportDownload{
		filePath:  filePath,
		year:      year,
		month:     month,
		expiresAt: time.Now().Add(ttl),
	}
	return token
}

func (s *exportDownloadStore) get(token string) (exportDownload, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked(time.Now())

	v, ok := s.items[token]
	if !ok {
		return exportDownload{}, false
	}
	if time.Now().After(v.expiresAt) {
		delete(s.items, token)
		return exportDownload{}, false
	}
	return v, true
}

func (s *exportDownloadStore) delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, token)
}

func (s *exportDownloadStore) purgeExpiredLocked(now time.Time) {
	for k, v := range s.items {
		if now.After(v.expiresAt) {
			delete(s.items, k)
		}
	}
}

func newRandomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
