package opendota

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHeroes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet ||
				r.URL.Path != "/api/heroes" {
				t.Errorf(
					"requisição inesperada: %s %s",
					r.Method,
					r.URL.Path,
				)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{
					"id": 1,
					"name": "npc_dota_hero_antimage",
					"localized_name": "Anti-Mage",
					"primary_attr": "agi",
					"attack_type": "Melee",
					"roles": ["Carry", "Escape"]
				}
			]`))
		},
	))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL + "/api"
	client.httpClient = server.Client()

	heroes, err := client.Heroes(context.Background())
	if err != nil {
		t.Fatalf("erro ao buscar heróis: %v", err)
	}

	if len(heroes) != 1 {
		t.Fatalf(
			"quantidade de heróis: %d; esperado: 1",
			len(heroes),
		)
	}

	hero := heroes[0]

	if hero.ID != 1 {
		t.Errorf("ID recebido: %d; esperado: 1", hero.ID)
	}

	if hero.Name != "npc_dota_hero_antimage" {
		t.Errorf(
			"nome interno recebido: %q; esperado: npc_dota_hero_antimage",
			hero.Name,
		)
	}

	if hero.LocalizedName != "Anti-Mage" {
		t.Errorf(
			"nome recebido: %q; esperado: Anti-Mage",
			hero.LocalizedName,
		)
	}

	if hero.PrimaryAttr != "agi" {
		t.Errorf(
			"atributo recebido: %q; esperado: agi",
			hero.PrimaryAttr,
		)
	}

	if hero.AttackType != "Melee" {
		t.Errorf(
			"tipo de ataque recebido: %q; esperado: Melee",
			hero.AttackType,
		)
	}

	if len(hero.Roles) != 2 {
		t.Fatalf(
			"quantidade de papéis: %d; esperado: 2",
			len(hero.Roles),
		)
	}

	if hero.Roles[0] != "Carry" || hero.Roles[1] != "Escape" {
		t.Errorf("papéis inesperados: %v", hero.Roles)
	}
}

func TestHeroesUsesCache(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{"id": 1, "localized_name": "Anti-Mage"}
			]`))
		},
	))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL
	client.httpClient = server.Client()

	for range 2 {
		heroes, err := client.Heroes(context.Background())
		if err != nil {
			t.Fatalf("erro ao buscar heróis: %v", err)
		}

		if len(heroes) != 1 {
			t.Fatalf(
				"quantidade de heróis: %d; esperado: 1",
				len(heroes),
			)
		}

		if heroes[0].ID != 1 {
			t.Fatalf("ID recebido: %d; esperado: 1", heroes[0].ID)
		}
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("chamadas externas: %d; esperado: 1", got)
	}
}

func TestHeroesRefreshesExpiredCache(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{"id": 1, "localized_name": "Anti-Mage"}
			]`))
		},
	))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL
	client.httpClient = server.Client()

	if _, err := client.Heroes(context.Background()); err != nil {
		t.Fatalf("erro na primeira consulta: %v", err)
	}

	client.heroesMu.Lock()
	client.heroesExpiresAt = time.Now().Add(-time.Second)
	client.heroesMu.Unlock()

	heroes, err := client.Heroes(context.Background())
	if err != nil {
		t.Fatalf("erro ao atualizar cache: %v", err)
	}

	if len(heroes) != 1 {
		t.Fatalf("quantidade de heróis: %d; esperado: 1", len(heroes))
	}

	if got := calls.Load(); got != 2 {
		t.Fatalf("chamadas externas: %d; esperado: 2", got)
	}

	if _, err := client.Heroes(context.Background()); err != nil {
		t.Fatalf("erro após atualizar cache: %v", err)
	}

	if got := calls.Load(); got != 2 {
		t.Fatalf(
			"chamadas externas após reutilizar cache: %d; esperado: 2",
			got,
		)
	}
}

func TestHeroesConcurrentCallsShareCache(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{"id": 1, "localized_name": "Anti-Mage"}
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

	for range queries {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			heroes, err := client.Heroes(context.Background())
			if err != nil {
				t.Errorf("erro ao buscar heróis: %v", err)
				return
			}

			if len(heroes) != 1 {
				t.Errorf(
					"quantidade de heróis: %d; esperado: 1",
					len(heroes),
				)
				return
			}

			if heroes[0].ID != 1 {
				t.Errorf("ID recebido: %d; esperado: 1", heroes[0].ID)
			}
		}()
	}

	close(start)
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("chamadas externas: %d; esperado: 1", got)
	}
}

func TestHeroesCacheIsNotModifiedByCaller(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{
					"id": 1,
					"localized_name": "Anti-Mage",
					"roles": ["Carry", "Escape"]
				}
			]`))
		},
	))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL
	client.httpClient = server.Client()

	heroes, err := client.Heroes(context.Background())
	if err != nil {
		t.Fatalf("erro na primeira consulta: %v", err)
	}

	if len(heroes) != 1 || len(heroes[0].Roles) != 2 {
		t.Fatalf("resposta inicial inesperada: %+v", heroes)
	}

	heroes[0].Roles[0] = "Modified"

	cachedHeroes, err := client.Heroes(context.Background())
	if err != nil {
		t.Fatalf("erro ao consultar cache: %v", err)
	}

	if len(cachedHeroes) != 1 || len(cachedHeroes[0].Roles) != 2 {
		t.Fatalf("resposta do cache inesperada: %+v", cachedHeroes)
	}

	if cachedHeroes[0].Roles[0] != "Carry" {
		t.Fatalf(
			"cache foi alterado pelo consumidor: papel recebido %q",
			cachedHeroes[0].Roles[0],
		)
	}
}
