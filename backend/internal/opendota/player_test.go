package opendota

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPlayerFetchesAndCachesByAccountID(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("User-Agent") != "AncientAnalytics/1.0" {
			t.Error("consulta deve usar GET e identificar a aplicação")
		}
		var id uint32
		if _, err := fmt.Sscanf(r.URL.Path, "/players/%d", &id); err != nil {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"rank_tier": 80, "leaderboard_rank": 123,
			"profile": map[string]any{"account_id": id, "personaname": "Player", "steamid": "76561198000000000", "avatarfull": "https://avatars.steamstatic.com/avatar.jpg"},
		})
	}))
	defer server.Close()
	client := NewClient()
	client.baseURL, client.httpClient = server.URL, server.Client()

	data, err := client.Player(context.Background(), 42)
	if err != nil || data.Profile == nil || data.Profile.AccountID != 42 || data.Profile.PersonaName != "Player" ||
		data.Profile.SteamID != "76561198000000000" || data.RankTier == nil || *data.RankTier != 80 {
		t.Fatalf("perfil inesperado: %+v; erro: %v", data, err)
	}
	data.Profile.PersonaName = "Modified"
	*data.RankTier = 0
	cached, err := client.Player(context.Background(), 42)
	if err != nil || cached.Profile.PersonaName != "Player" || *cached.RankTier != 80 || calls.Load() != 1 {
		t.Fatalf("cache não foi preservado: %+v; erro: %v; consultas: %d", cached, err, calls.Load())
	}
	other, err := client.Player(context.Background(), 43)
	if err != nil || other.Profile.AccountID != 43 || calls.Load() != 2 {
		t.Fatal("contas diferentes precisam de entradas independentes no cache")
	}
	client.playersMu.Lock()
	entry := client.playersCache[42]
	entry.expiresAt = time.Now().Add(-time.Second)
	client.playersCache[42] = entry
	client.playersMu.Unlock()
	if _, err := client.Player(context.Background(), 42); err != nil || calls.Load() != 3 {
		t.Fatalf("perfil vencido deveria ser atualizado: %v", err)
	}
}

func TestPlayerUnavailableAndInvalidResponses(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   int
		body     string
		notFound bool
	}{
		{"http_404", 404, `{}`, true},
		{"perfil_indisponivel", 200, `{"profile":null,"rank_tier":null}`, true},
		{"http_503", 503, `{}`, false},
		{"http_403", 403, `{}`, false},
		{"json_invalido", 200, `{"profile":`, false},
		{"resposta_nula", 200, `null`, false},
		{"conta_incorreta", 200, `{"profile":{"account_id":43}}`, false},
		{"rank_negativo", 200, `{"profile":{"account_id":42},"rank_tier":-1}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			client := NewClient()
			client.baseURL, client.httpClient = server.URL, server.Client()
			for range 2 {
				_, err := client.Player(context.Background(), 42)
				if err == nil || errors.Is(err, ErrPlayerNotFound) != test.notFound {
					t.Fatalf("erro inesperado: %v; esperado indisponível: %t", err, test.notFound)
				}
			}
			if calls.Load() != 2 {
				t.Fatal("erros não devem preencher o cache")
			}
		})
	}
}

func TestPlayerConcurrentQueriesAndCacheLimit(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"profile":{"account_id":42},"rank_tier":null,"leaderboard_rank":null}`))
	}))
	defer server.Close()
	client := NewClient()
	client.baseURL, client.httpClient = server.URL, server.Client()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Player(ctx, 42); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelamento não preservado: %v", err)
	}
	if _, err := client.Player(context.Background(), 0); err == nil || calls.Load() != 0 {
		t.Fatal("ID inválido ou consulta cancelada não deve acessar a fonte")
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := client.Player(context.Background(), 42); err != nil {
				t.Error(err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("consultas: %d; esperado: 1", calls.Load())
	}
	client.playersMu.Lock()
	for id := uint32(100); id < 100+playerCacheLimit; id++ {
		client.playersCache[id] = playerCacheEntry{expiresAt: time.Now().Add(time.Minute)}
	}
	delete(client.playersCache, 42)
	client.playersMu.Unlock()
	if _, err := client.Player(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	client.playersMu.Lock()
	defer client.playersMu.Unlock()
	if len(client.playersCache) > playerCacheLimit {
		t.Fatalf("cache sem limite: %d", len(client.playersCache))
	}
}
