package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestMetaServiceCachesHeroesAndExposesHealthEndpoints(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/heroStats" {
			t.Fatalf("upstream path = %s, want /heroStats", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"id":1,"localized_name":"Anti-Mage"}]`))
	}))
	defer upstream.Close()

	handler := NewHandler(Config{OpenDotaURL: upstream.URL, CacheTTL: time.Minute, HTTPClient: upstream.Client()})
	for range 2 {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/meta/heroes", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("heroes status = %d, want %d", response.Code, http.StatusOK)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("upstream calls = %d, want 1", calls.Load())
	}

	ready := httptest.NewRecorder()
	handler.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusOK {
		t.Fatalf("readyz status = %d, want %d", ready.Code, http.StatusOK)
	}
}
