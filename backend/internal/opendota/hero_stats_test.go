package opendota

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHeroStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/api/heroStats" {
				t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{
					"id": 1,
					"localized_name": "Anti-Mage",
					"pro_pick": 100,
					"pro_win": 60
				}
			]`))
		},
	))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL + "/api"
	client.httpClient = server.Client()

	stats, err := client.HeroStats(context.Background())
	if err != nil {
		t.Fatalf("erro ao buscar estatísticas: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("quantidade de estatísticas: %d; esperado: 1", len(stats))
	}

	stat := stats[0]
	if stat.ID != 1 {
		t.Errorf("ID recebido: %d; esperado: 1", stat.ID)
	}
	if stat.LocalizedName != "Anti-Mage" {
		t.Errorf("nome recebido: %q; esperado: Anti-Mage", stat.LocalizedName)
	}
	if stat.ProPick != 100 {
		t.Errorf("escolhas recebidas: %d; esperado: 100", stat.ProPick)
	}
	if stat.ProWin != 60 {
		t.Errorf("vitórias recebidas: %d; esperado: 60", stat.ProWin)
	}
}

func TestHeroStatsReturnsError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "HTTP_503",
			status: http.StatusServiceUnavailable,
			body:   `[]`,
		},
		{
			name:   "JSON_invalido",
			status: http.StatusOK,
			body:   `{"id":`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(test.status)
					w.Write([]byte(test.body))
				},
			))
			defer server.Close()

			client := NewClient()
			client.baseURL = server.URL
			client.httpClient = server.Client()

			_, err := client.HeroStats(context.Background())
			if err == nil {
				t.Fatal("esperado erro ao consultar estatísticas")
			}
		})
	}
}

func TestHeroStatsUsesCache(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{
					"id": 1,
					"localized_name": "Anti-Mage",
					"pro_pick": 8,
					"pro_win": 3
				}
			]`))
		},
	))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL
	client.httpClient = server.Client()

	for range 2 {
		stats, err := client.HeroStats(context.Background())
		if err != nil {
			t.Fatalf("erro ao buscar estatísticas: %v", err)
		}

		if len(stats) != 1 {
			t.Fatalf(
				"quantidade de estatísticas: %d; esperado: 1",
				len(stats),
			)
		}

		if stats[0].ProPick != 8 || stats[0].ProWin != 3 {
			t.Errorf("estatísticas inesperadas: %+v", stats[0])
		}
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("chamadas externas: %d; esperado: 1", got)
	}
}

func TestHeroStatsRefreshesExpiredCache(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			number := calls.Add(1)

			w.Header().Set("Content-Type", "application/json")

			_ = json.NewEncoder(w).Encode([]HeroStat{
				{
					ID:            1,
					LocalizedName: "Anti-Mage",
					ProPick:       int(number) * 8,
					ProWin:        3,
				},
			})
		},
	))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL
	client.httpClient = server.Client()

	stats, err := client.HeroStats(context.Background())
	if err != nil {
		t.Fatalf("erro na primeira consulta: %v", err)
	}

	if len(stats) != 1 || stats[0].ProPick != 8 {
		t.Fatalf("resposta inicial inesperada: %+v", stats)
	}

	// Simula um cache vencido.
	client.statsMu.Lock()
	client.statsExpiresAt = time.Now().Add(-time.Second)
	client.statsMu.Unlock()

	stats, err = client.HeroStats(context.Background())
	if err != nil {
		t.Fatalf("erro ao atualizar cache: %v", err)
	}

	if len(stats) != 1 || stats[0].ProPick != 16 {
		t.Fatalf("esperadas estatísticas atualizadas: %+v", stats)
	}

	// A terceira consulta deve reutilizar os dados atualizados.
	stats, err = client.HeroStats(context.Background())
	if err != nil {
		t.Fatalf("erro após atualizar cache: %v", err)
	}

	if len(stats) != 1 || stats[0].ProPick != 16 {
		t.Fatalf("cache atualizado não foi preservado: %+v", stats)
	}

	if got := calls.Load(); got != 2 {
		t.Fatalf("chamadas externas: %d; esperado: 2", got)
	}
}

func TestHeroStatsConcurrentCallsShareCache(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{
					"id": 1,
					"localized_name": "Anti-Mage",
					"pro_pick": 8,
					"pro_win": 3
				}
			]`))
		},
	))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL
	client.httpClient = server.Client()

	const queries = 10

	var wg sync.WaitGroup
	start := make(chan struct{})

	wg.Add(queries)

	for range queries {
		go func() {
			defer wg.Done()

			<-start

			stats, err := client.HeroStats(context.Background())
			if err != nil {
				t.Errorf("erro ao buscar estatísticas: %v", err)
				return
			}

			if len(stats) != 1 {
				t.Errorf(
					"quantidade de estatísticas: %d; esperado: 1",
					len(stats),
				)
				return
			}

			if stats[0].ProPick != 8 || stats[0].ProWin != 3 {
				t.Errorf("estatísticas inesperadas: %+v", stats[0])
			}
		}()
	}

	close(start)
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("chamadas externas: %d; esperado: 1", got)
	}
}
