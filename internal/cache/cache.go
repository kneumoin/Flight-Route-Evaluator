package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kneumoin/nepal/internal/config"
)

type Store struct {
	cfg   config.CacheConfig
	ttl   time.Duration
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func New(cfg config.CacheConfig) (*Store, error) {
	ttl, err := cfg.TTLDuration()
	if err != nil {
		return nil, err
	}
	return &Store{cfg: cfg, ttl: ttl, locks: make(map[string]*sync.Mutex)}, nil
}

func (s *Store) keyHash(provider, key string) string {
	h := sha256.Sum256([]byte(provider + "|" + key))
	return hex.EncodeToString(h[:])
}

func (s *Store) path(provider, hash string) string {
	return filepath.Join(s.cfg.Directory, provider, hash+".json")
}

func (s *Store) Get(provider, key string) ([]byte, bool, error) {
	if !s.cfg.Enabled {
		return nil, false, nil
	}
	p := s.path(provider, s.keyHash(provider, key))
	st, err := os.Stat(p)
	if err != nil {
		return nil, false, nil
	}
	if time.Since(st.ModTime()) > s.ttl {
		return nil, false, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false, nil
	}
	if !json.Valid(data) {
		return nil, false, nil
	}
	return data, true, nil
}

func (s *Store) Put(provider, key string, data []byte) error {
	if !s.cfg.Enabled {
		return nil
	}
	dir := filepath.Join(s.cfg.Directory, provider)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path(provider, s.keyHash(provider, key)), data, 0o644)
}

func (s *Store) Fetch(provider, key string, fetch func() ([]byte, error)) ([]byte, error) {
	l := s.lockFor(provider + key)
	l.Lock()
	defer l.Unlock()

	if data, ok, _ := s.Get(provider, key); ok {
		return data, nil
	}
	data, err := fetch()
	if err != nil {
		return nil, err
	}
	if err := s.Put(provider, key, data); err != nil {
		return data, fmt.Errorf("cache put: %w", err)
	}
	return data, nil
}

func (s *Store) lockFor(k string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locks[k] == nil {
		s.locks[k] = &sync.Mutex{}
	}
	return s.locks[k]
}
