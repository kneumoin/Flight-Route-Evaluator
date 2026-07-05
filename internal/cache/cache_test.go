package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kneumoin/nepal/internal/config"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := New(config.CacheConfig{Enabled: true, TTL: "1h", Directory: dir})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCache_MissAndHit(t *testing.T) {
	s := testStore(t)
	_, ok, _ := s.Get("aviasales", "k1")
	if ok {
		t.Fatal("expected miss")
	}
	if err := s.Put("aviasales", "k1", []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	data, ok, err := s.Get("aviasales", "k1")
	if err != nil || !ok || string(data) != `{"a":1}` {
		t.Fatalf("hit failed: %v %v %s", ok, err, data)
	}
}

func TestCache_Expired(t *testing.T) {
	s := testStore(t)
	s.ttl = time.Millisecond
	_ = s.Put("aviasales", "k1", []byte(`{}`))
	time.Sleep(5 * time.Millisecond)
	_, ok, _ := s.Get("aviasales", "k1")
	if ok {
		t.Fatal("expected expired miss")
	}
}

func TestCache_Fetch(t *testing.T) {
	s := testStore(t)
	calls := 0
	data, err := s.Fetch("aviasales", "key1", func() ([]byte, error) {
		calls++
		return []byte(`{"ok":true}`), nil
	})
	if err != nil || string(data) != `{"ok":true}` || calls != 1 {
		t.Fatalf("first fetch failed calls=%d", calls)
	}
	_, err = s.Fetch("aviasales", "key1", func() ([]byte, error) {
		calls++
		return []byte(`{"ok":false}`), nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("expected cache hit, calls=%d", calls)
	}
}

func TestCache_CorruptedJSON(t *testing.T) {
	s := testStore(t)
	p := s.path("aviasales", s.keyHash("aviasales", "k1"))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`not-json`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, ok, _ := s.Get("aviasales", "k1")
	if ok {
		t.Fatal("corrupt should miss")
	}
}
